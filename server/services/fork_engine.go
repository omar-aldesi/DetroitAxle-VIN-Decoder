package services

import (
	"sort"
	"strings"

	"main/models"
)

// NormalizeForkValue trims, collapses internal whitespace, and upper-cases, so "JP9" and
// "jp9 " are treated as equal and never spawn a phantom fork. Display uses the raw value.
func NormalizeForkValue(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}

// RecordPoint ingests one verified VIN sighting. See the algorithm in
// docs/fork-range-system.md §6.1.
func (e *ForkEngine) RecordPoint(in PointInput) (ForkOutcome, error) {
	if !in.Verified {
		return OutcomeIgnored, nil
	}
	ok, err := e.Store.IsForkField(in.Make, in.Model, in.FieldKey)
	if err != nil {
		return "", err
	}
	if !ok {
		return OutcomeNotForkField, nil
	}

	norm := NormalizeForkValue(in.Value)

	ranges, err := e.Store.RangesFor(in.BuildKey, in.FieldKey)
	if err != nil {
		return "", err
	}
	cov := coveringRange(ranges, in.Serial)

	// Covered by a same-value range → reinforce the assumption.
	if cov != nil && cov.ValueNorm == norm {
		cov.Observations++
		if in.Serial < cov.SerialMinSeen {
			cov.SerialMinSeen = in.Serial
		}
		if in.Serial > cov.SerialMaxSeen {
			cov.SerialMaxSeen = in.Serial
		}
		cov.UpdatedAt = e.now()
		if err := e.Store.SaveRange(cov); err != nil {
			return "", err
		}
		if err := e.absorbCoveredAgreeingPoints(in.BuildKey, in.FieldKey); err != nil {
			return "", err
		}
		e.emit(ForkEvent{in.BuildKey, in.FieldKey, in.Serial, OutcomeReinforced})
		return OutcomeReinforced, nil
	}

	// Otherwise hold this as a pending point, then try to form/carve a range.
	if err := e.Store.SavePoint(&models.FieldPoint{
		BuildKey: in.BuildKey, FieldKey: in.FieldKey, Serial: in.Serial,
		Value: in.Value, ValueNorm: norm, Source: in.Source, Actor: in.ActorID,
		CreatedAt: e.now(),
	}); err != nil {
		return "", err
	}

	points, err := e.Store.PointsFor(in.BuildKey, in.FieldKey)
	if err != nil {
		return "", err
	}
	agreeing := pointsWithValue(points, norm)

	if cov != nil {
		// Contradiction inside an existing range → need >=2 agreeing exception points
		// inside this range to carve an inner range.
		inside := pointsInRange(agreeing, cov)
		if len(inside) >= 2 {
			lo, hi := minMaxSerial(inside)
			if err := e.carve(cov, lo, hi, len(inside), in, norm); err != nil {
				return "", err
			}
			if err := e.Store.DeletePoints(pointIDs(inside)); err != nil {
				return "", err
			}
			e.emit(ForkEvent{in.BuildKey, in.FieldKey, in.Serial, OutcomeForked})
			return OutcomeForked, nil
		}
		return OutcomePending, nil
	}

	// No covering range → form new ranges from agreeing uncovered points, grouped by gap
	// segment so a formed range never straddles an existing range. Each gap needs its own
	// two agreeing sightings.
	uncovered := pointsUncovered(agreeing, ranges)
	created := false
	for _, group := range groupBySegment(uncovered, ranges) {
		if len(group) < 2 {
			continue
		}
		lo, hi := minMaxSerial(group)
		end := hi
		nr := &models.FieldRange{
			BuildKey: in.BuildKey, FieldKey: in.FieldKey,
			SerialStart: lo, SerialEnd: &end,
			Value: in.Value, ValueNorm: norm, Origin: "vin", Source: in.Source,
			Observations: len(group), SerialMinSeen: lo, SerialMaxSeen: hi,
			BoundaryExact: false, LastActor: in.ActorID,
			CreatedAt: e.now(), UpdatedAt: e.now(),
		}
		if err := e.Store.SaveRange(nr); err != nil {
			return "", err
		}
		if err := e.Store.DeletePoints(pointIDs(group)); err != nil {
			return "", err
		}
		created = true
	}
	if created {
		e.emit(ForkEvent{in.BuildKey, in.FieldKey, in.Serial, OutcomeRangeCreated})
		return OutcomeRangeCreated, nil
	}

	return OutcomePending, nil
}

// absorbCoveredAgreeingPoints deletes pending points whose serial is now covered by a
// range with the same normalized value — they are redundant evidence, not contradictions.
// Disagreeing points are kept; they may carve later.
func (e *ForkEngine) absorbCoveredAgreeingPoints(buildKey, fieldKey string) error {
	points, err := e.Store.PointsFor(buildKey, fieldKey)
	if err != nil {
		return err
	}
	if len(points) == 0 {
		return nil
	}
	ranges, err := e.Store.RangesFor(buildKey, fieldKey)
	if err != nil {
		return err
	}
	var ids []uint
	for _, p := range points {
		if cov := coveringRange(ranges, p.Serial); cov != nil && cov.ValueNorm == p.ValueNorm {
			ids = append(ids, p.ID)
		}
	}
	return e.Store.DeletePoints(ids)
}

// carve splits range r into up to three pieces: [r.start, lo-1]=old, [lo, hi]=new,
// [hi+1, r.end]=old. Empty pieces are omitted. Append-only: never merges, never rewrites
// below the boundary; all introduced boundaries are approximate. Remnant pieces keep only
// the evidence that actually falls inside them.
func (e *ForkEngine) carve(r *models.FieldRange, lo, hi int64, obs int, in PointInput, norm string) error {
	origEnd := r.SerialEnd
	origVal, origNorm := r.Value, r.ValueNorm
	origOrigin, origSource := r.Origin, r.Source
	origObs := r.Observations
	origMin, origMax := r.SerialMinSeen, r.SerialMaxSeen
	now := e.now()

	// Left piece: shrink r, or delete it if the inner range starts at r's own start.
	if lo > r.SerialStart {
		leftEnd := lo - 1
		r.SerialEnd = &leftEnd
		r.SerialMinSeen, r.SerialMaxSeen, r.Observations =
			remnantEvidence(r.SerialStart, &leftEnd, origMin, origMax, origObs)
		r.BoundaryExact = false
		r.UpdatedAt = now
		if err := e.Store.SaveRange(r); err != nil {
			return err
		}
	} else {
		if err := e.Store.DeleteRange(r.ID); err != nil {
			return err
		}
	}

	// Inner piece (the new value).
	innerEnd := hi
	if err := e.Store.SaveRange(&models.FieldRange{
		BuildKey: in.BuildKey, FieldKey: in.FieldKey,
		SerialStart: lo, SerialEnd: &innerEnd,
		Value: in.Value, ValueNorm: norm, Origin: "vin", Source: in.Source,
		Observations: obs, SerialMinSeen: lo, SerialMaxSeen: hi,
		BoundaryExact: false, LastActor: in.ActorID,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return err
	}

	// Right piece (the old value continues), if anything remains above the inner range.
	if origEnd == nil || hi < *origEnd {
		rightStart := hi + 1
		mn, mx, ro := remnantEvidence(rightStart, origEnd, origMin, origMax, origObs)
		if err := e.Store.SaveRange(&models.FieldRange{
			BuildKey: in.BuildKey, FieldKey: in.FieldKey,
			SerialStart: rightStart, SerialEnd: origEnd,
			Value: origVal, ValueNorm: origNorm, Origin: origOrigin, Source: origSource,
			Observations: ro, SerialMinSeen: mn, SerialMaxSeen: mx,
			BoundaryExact: false, LastActor: in.ActorID,
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RecordRange writes an explicit human span. Manual ranges are authoritative: overlapping
// ranges are trimmed/removed so the manual one wins. No two-point rule.
func (e *ForkEngine) RecordRange(in RangeInput) (ForkOutcome, error) {
	if !in.Verified {
		return OutcomeIgnored, nil
	}
	ok, err := e.Store.IsForkField(in.Make, in.Model, in.FieldKey)
	if err != nil {
		return "", err
	}
	if !ok {
		return OutcomeNotForkField, nil
	}
	norm := NormalizeForkValue(in.Value)
	now := e.now()

	ranges, err := e.Store.RangesFor(in.BuildKey, in.FieldKey)
	if err != nil {
		return "", err
	}

	start := in.SerialStart
	end := openEnd
	if in.SerialEnd != nil {
		end = *in.SerialEnd
	}

	// Trim/remove any existing range overlapping [start, end] — manual wins.
	for i := range ranges {
		r := ranges[i]
		rEnd := openEnd
		if r.SerialEnd != nil {
			rEnd = *r.SerialEnd
		}
		if r.SerialStart > end || rEnd < start {
			continue // disjoint
		}
		if err := e.Store.DeleteRange(r.ID); err != nil {
			return "", err
		}
		// Left remnant of the old range, below the manual span.
		if r.SerialStart < start {
			le := start - 1
			if err := e.Store.SaveRange(cloneRangeSpan(&r, r.SerialStart, &le, now)); err != nil {
				return "", err
			}
		}
		// Right remnant, above the manual span (only possible when manual end is finite).
		if in.SerialEnd != nil && rEnd > end {
			rs := end + 1
			var rEndPtr *int64
			if r.SerialEnd != nil {
				rEndPtr = r.SerialEnd
			}
			if err := e.Store.SaveRange(cloneRangeSpan(&r, rs, rEndPtr, now)); err != nil {
				return "", err
			}
		}
	}

	maxSeen := start
	if in.SerialEnd != nil {
		maxSeen = *in.SerialEnd
	}
	if err := e.Store.SaveRange(&models.FieldRange{
		BuildKey: in.BuildKey, FieldKey: in.FieldKey,
		SerialStart: start, SerialEnd: in.SerialEnd,
		Value: in.Value, ValueNorm: norm, Origin: "manual", Source: in.Source,
		Observations: 1, SerialMinSeen: start, SerialMaxSeen: maxSeen,
		BoundaryExact: true, LastActor: in.ActorID,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return "", err
	}
	// Pending points the manual range now agrees with are absorbed; disagreeing
	// points are kept (newer evidence may carve the manual range later).
	if err := e.absorbCoveredAgreeingPoints(in.BuildKey, in.FieldKey); err != nil {
		return "", err
	}
	e.emit(ForkEvent{in.BuildKey, in.FieldKey, start, OutcomeManualRange})
	return OutcomeManualRange, nil
}

// RecordBaseRange writes a build-key-wide base value WITHOUT destroying existing range
// data: it fills only the serial gaps not covered by any existing range. Used for
// verified build-key-wide edits (no serial known) and legacy backfill, where the value is
// authoritative for the build key as a whole but must never erase observed per-serial
// forks. Safe to re-run: fully covered fields are a no-op.
func (e *ForkEngine) RecordBaseRange(in RangeInput) (ForkOutcome, error) {
	if !in.Verified {
		return OutcomeIgnored, nil
	}
	ok, err := e.Store.IsForkField(in.Make, in.Model, in.FieldKey)
	if err != nil {
		return "", err
	}
	if !ok {
		return OutcomeNotForkField, nil
	}
	norm := NormalizeForkValue(in.Value)
	now := e.now()

	ranges, err := e.Store.RangesFor(in.BuildKey, in.FieldKey)
	if err != nil {
		return "", err
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].SerialStart < ranges[j].SerialStart })

	saveGap := func(start int64, end *int64) error {
		maxSeen := start
		if end != nil {
			maxSeen = *end
		}
		return e.Store.SaveRange(&models.FieldRange{
			BuildKey: in.BuildKey, FieldKey: in.FieldKey,
			SerialStart: start, SerialEnd: end,
			Value: in.Value, ValueNorm: norm, Origin: "manual", Source: in.Source,
			Observations: 1, SerialMinSeen: start, SerialMaxSeen: maxSeen,
			BoundaryExact: false, LastActor: in.ActorID,
			CreatedAt: now, UpdatedAt: now,
		})
	}

	cur := int64(0) // next uncovered serial
	for _, r := range ranges {
		if r.SerialStart > cur {
			gapEnd := r.SerialStart - 1
			if err := saveGap(cur, &gapEnd); err != nil {
				return "", err
			}
		}
		if r.SerialEnd == nil {
			cur = openEnd
			break
		}
		if *r.SerialEnd+1 > cur {
			cur = *r.SerialEnd + 1
		}
	}
	if cur != openEnd {
		if err := saveGap(cur, nil); err != nil {
			return "", err
		}
	}

	if err := e.absorbCoveredAgreeingPoints(in.BuildKey, in.FieldKey); err != nil {
		return "", err
	}
	e.emit(ForkEvent{in.BuildKey, in.FieldKey, in.SerialStart, OutcomeBaseRange})
	return OutcomeBaseRange, nil
}
