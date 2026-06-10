package services

import (
	"sort"
	"time"

	"main/models"
)

func coveringRange(ranges []models.FieldRange, serial int64) *models.FieldRange {
	var best *models.FieldRange
	for i := range ranges {
		r := &ranges[i]
		if r.SerialStart <= serial && (r.SerialEnd == nil || serial <= *r.SerialEnd) {
			if best == nil || r.SerialStart > best.SerialStart {
				best = r
			}
		}
	}
	return best
}

func spanOverlapsAny(ranges []models.FieldRange, start, end int64) bool {
	for _, r := range ranges {
		rEnd := openEnd
		if r.SerialEnd != nil {
			rEnd = *r.SerialEnd
		}
		if r.SerialStart <= end && start <= rEnd {
			return true
		}
	}
	return false
}

func pointsWithValue(points []models.FieldPoint, norm string) []models.FieldPoint {
	var out []models.FieldPoint
	for _, p := range points {
		if p.ValueNorm == norm {
			out = append(out, p)
		}
	}
	return out
}

func pointsInRange(points []models.FieldPoint, r *models.FieldRange) []models.FieldPoint {
	var out []models.FieldPoint
	for _, p := range points {
		if p.Serial >= r.SerialStart && (r.SerialEnd == nil || p.Serial <= *r.SerialEnd) {
			out = append(out, p)
		}
	}
	return out
}

func pointsUncovered(points []models.FieldPoint, ranges []models.FieldRange) []models.FieldPoint {
	var out []models.FieldPoint
	for _, p := range points {
		if coveringRange(ranges, p.Serial) == nil {
			out = append(out, p)
		}
	}
	return out
}

// groupBySegment splits serial-sorted points into groups separated by existing ranges.
// Points in the same gap (no range between them) belong to one group; a range between two
// points forces a new group so a formed range never spans across an existing one.
func groupBySegment(points []models.FieldPoint, ranges []models.FieldRange) [][]models.FieldPoint {
	sorted := make([]models.FieldPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Serial < sorted[j].Serial })

	var groups [][]models.FieldPoint
	for _, p := range sorted {
		if n := len(groups); n > 0 {
			last := groups[n-1]
			prev := last[len(last)-1]
			if !spanOverlapsAny(ranges, prev.Serial, p.Serial) {
				groups[n-1] = append(last, p)
				continue
			}
		}
		groups = append(groups, []models.FieldPoint{p})
	}
	return groups
}

// remnantEvidence computes honest evidence bounds for a remnant [start, end] cut out of a
// range whose evidence spanned [mn, mx] with obs observations. Only min/max are stored, so
// the only provable observations are the two extremes:
//   - whole seen span inside the remnant → keep everything;
//   - one extreme inside → that single provable observation;
//   - nothing provable inside → a nominal single observation at the remnant start.
//
// Resolution then degrades to assumed/extrapolated instead of reporting "observed" for
// serials nobody actually saw. Future reinforcements rebuild the count organically.
func remnantEvidence(start int64, end *int64, mn, mx int64, obs int) (int64, int64, int) {
	hi := openEnd
	if end != nil {
		hi = *end
	}
	mnIn := mn >= start && mn <= hi
	mxIn := mx >= start && mx <= hi
	switch {
	case mnIn && mxIn:
		return mn, mx, obs
	case mnIn:
		return mn, mn, 1
	case mxIn:
		return mx, mx, 1
	default:
		return start, start, 1
	}
}

func minMaxSerial(points []models.FieldPoint) (int64, int64) {
	lo, hi := points[0].Serial, points[0].Serial
	for _, p := range points[1:] {
		if p.Serial < lo {
			lo = p.Serial
		}
		if p.Serial > hi {
			hi = p.Serial
		}
	}
	return lo, hi
}

func pointIDs(points []models.FieldPoint) []uint {
	ids := make([]uint, len(points))
	for i, p := range points {
		ids[i] = p.ID
	}
	return ids
}

func cloneRangeSpan(src *models.FieldRange, start int64, end *int64, now time.Time) *models.FieldRange {
	mn, mx, obs := remnantEvidence(start, end, src.SerialMinSeen, src.SerialMaxSeen, src.Observations)
	return &models.FieldRange{
		BuildKey: src.BuildKey, FieldKey: src.FieldKey,
		SerialStart: start, SerialEnd: end,
		Value: src.Value, ValueNorm: src.ValueNorm, Origin: src.Origin, Source: src.Source,
		Observations: obs, SerialMinSeen: mn, SerialMaxSeen: mx,
		BoundaryExact: false, LastActor: src.LastActor,
		CreatedAt: now, UpdatedAt: now,
	}
}
