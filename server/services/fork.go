package services

// fork.go — build-number-tier fork/range engine.
//
// Two write doors (RecordPoint, RecordRange) and two read doors (Resolve,
// ResolveByBuildKey). All data access goes through ForkStore so the algorithm can be
// unit-tested with an in-memory fake. See docs/fork-range-system.md.

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"main/models"

	"gorm.io/gorm"
)

const openEnd = int64(math.MaxInt64) // sentinel for an open-ended range during comparisons

// ForkOutcome reports what a write did.
type ForkOutcome string

const (
	OutcomeIgnored      ForkOutcome = "ignored"        // unverified — rule #2
	OutcomeNotForkField ForkOutcome = "not_fork_field" // caller should treat as a build-key field
	OutcomePending      ForkOutcome = "pending"        // held; not yet a range (rule #1)
	OutcomeReinforced   ForkOutcome = "reinforced"
	OutcomeRangeCreated ForkOutcome = "range_created"
	OutcomeForked       ForkOutcome = "forked"
	OutcomeManualRange  ForkOutcome = "manual_range"
)

// PointInput is a single verified VIN sighting.
type PointInput struct {
	BuildKey string
	Serial   int64
	FieldKey string
	Value    string
	Source   string
	Verified bool
	ActorID  *uint
	Make     string
	Model    string
}

// RangeInput is an explicit human/manual span. SerialEnd nil = open-ended.
type RangeInput struct {
	BuildKey    string
	SerialStart int64
	SerialEnd   *int64
	FieldKey    string
	Value       string
	Source      string
	Verified    bool
	ActorID     *uint
	Make        string
	Model       string
}

// ForkEvent is published on every confirming write (optional subscribers).
type ForkEvent struct {
	BuildKey string
	FieldKey string
	Serial   int64
	Outcome  ForkOutcome
}

// ForkStore is the data-access surface the engine needs.
type ForkStore interface {
	IsForkField(make_, model, fieldKey string) (bool, error)
	RangesFor(buildKey, fieldKey string) ([]models.FieldRange, error)
	PointsFor(buildKey, fieldKey string) ([]models.FieldPoint, error)
	SaveRange(r *models.FieldRange) error // insert (ID==0) or update
	DeleteRange(id uint) error
	SavePoint(p *models.FieldPoint) error
	DeletePoints(ids []uint) error
}

// ForkEngine holds the fork logic. Now and Emit are injectable for testing.
type ForkEngine struct {
	Store ForkStore
	Now   func() time.Time
	Emit  func(ForkEvent)
}

func NewForkEngine(store ForkStore) *ForkEngine {
	return &ForkEngine{Store: store, Now: time.Now}
}

func (e *ForkEngine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *ForkEngine) emit(ev ForkEvent) {
	if e.Emit != nil {
		e.Emit(ev)
	}
}

// ─── normalization ────────────────────────────────────────────────────────────

// NormalizeForkValue trims, collapses internal whitespace, and upper-cases, so "JP9" and
// "jp9 " are treated as equal and never spawn a phantom fork. Display uses the raw value.
func NormalizeForkValue(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}

// ─── RecordPoint ─────────────────────────────────────────────────────────────

// RecordPoint ingests one verified VIN sighting. See the algorithm in
// docs/fork-range-system.md §6.1.
func (e *ForkEngine) RecordPoint(in PointInput) (ForkOutcome, error) {
	if !in.Verified {
		return OutcomeIgnored, nil // rule #2
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
			if err := e.carve(cov, lo, hi, in, norm); err != nil {
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

	// No covering range → form a brand-new range from >=2 agreeing uncovered points.
	uncovered := pointsUncovered(agreeing, ranges)
	if len(uncovered) >= 2 {
		lo, hi := minMaxSerial(uncovered)
		if spanOverlapsAny(ranges, lo, hi) {
			return OutcomePending, nil // straddles a boundary — leave for reconciliation
		}
		end := hi
		nr := &models.FieldRange{
			BuildKey: in.BuildKey, FieldKey: in.FieldKey,
			SerialStart: lo, SerialEnd: &end,
			Value: in.Value, ValueNorm: norm, Origin: "vin", Source: in.Source,
			Observations: len(uncovered), SerialMinSeen: lo, SerialMaxSeen: hi,
			BoundaryExact: false, LastActor: in.ActorID,
			CreatedAt: e.now(), UpdatedAt: e.now(),
		}
		if err := e.Store.SaveRange(nr); err != nil {
			return "", err
		}
		if err := e.Store.DeletePoints(pointIDs(uncovered)); err != nil {
			return "", err
		}
		e.emit(ForkEvent{in.BuildKey, in.FieldKey, in.Serial, OutcomeRangeCreated})
		return OutcomeRangeCreated, nil
	}

	return OutcomePending, nil
}

// carve splits range r into up to three pieces: [r.start, lo-1]=old, [lo, hi]=new,
// [hi+1, r.end]=old. Empty pieces are omitted. Append-only: never merges, never rewrites
// below the boundary; all introduced boundaries are approximate.
func (e *ForkEngine) carve(r *models.FieldRange, lo, hi int64, in PointInput, norm string) error {
	origEnd := r.SerialEnd
	origVal, origNorm := r.Value, r.ValueNorm
	origOrigin, origSource := r.Origin, r.Source
	origObs := r.Observations
	now := e.now()

	// Left piece: shrink r, or delete it if the inner range starts at r's own start.
	if lo > r.SerialStart {
		leftEnd := lo - 1
		r.SerialEnd = &leftEnd
		if r.SerialMaxSeen > leftEnd {
			r.SerialMaxSeen = leftEnd
		}
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
		Observations: 2, SerialMinSeen: lo, SerialMaxSeen: hi,
		BoundaryExact: false, LastActor: in.ActorID,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return err
	}

	// Right piece (the old value continues), if anything remains above the inner range.
	if origEnd == nil || hi < *origEnd {
		rightStart := hi + 1
		rightMax := rightStart
		if origEnd != nil {
			rightMax = *origEnd
		}
		if err := e.Store.SaveRange(&models.FieldRange{
			BuildKey: in.BuildKey, FieldKey: in.FieldKey,
			SerialStart: rightStart, SerialEnd: origEnd,
			Value: origVal, ValueNorm: origNorm, Origin: origOrigin, Source: origSource,
			Observations: origObs, SerialMinSeen: rightStart, SerialMaxSeen: rightMax,
			BoundaryExact: false, LastActor: in.ActorID,
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ─── RecordRange (manual) ───────────────────────────────────────────────────

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
	e.emit(ForkEvent{in.BuildKey, in.FieldKey, start, OutcomeManualRange})
	return OutcomeManualRange, nil
}

// ─── Resolve (read) ─────────────────────────────────────────────────────────

// ConfidenceTier describes how trustworthy a resolved value is.
type ConfidenceTier string

const (
	ConfObserved     ConfidenceTier = "observed"     // within evidence of a confirmed range
	ConfAssumed      ConfidenceTier = "assumed"       // within range, thin evidence
	ConfExtrapolated ConfidenceTier = "extrapolated" // beyond observed evidence
	ConfManual       ConfidenceTier = "manual"        // human-entered
)

// Resolved is a single field's value for a VIN.
type Resolved struct {
	Value         string
	Source        string
	Origin        string
	Confidence    ConfidenceTier
	Observations  int
	SerialMinSeen int64
	SerialMaxSeen int64
	RangeID       uint
}

// RangeView is one confirmed range in the build-key view.
type RangeView struct {
	SerialStart   int64
	SerialEnd     *int64
	Value         string
	Origin        string
	Source        string
	Observations  int
	BoundaryExact bool
}

// Resolve returns the value of every fork field for one VIN serial.
func (e *ForkEngine) Resolve(buildKey string, serial int64, fieldKeys []string) (map[string]Resolved, error) {
	out := make(map[string]Resolved, len(fieldKeys))
	for _, fk := range fieldKeys {
		ranges, err := e.Store.RangesFor(buildKey, fk)
		if err != nil {
			return nil, err
		}
		cov := coveringRange(ranges, serial)
		if cov == nil {
			continue // unknown for this field
		}
		out[fk] = Resolved{
			Value: cov.Value, Source: cov.Source, Origin: cov.Origin,
			Confidence:    confidenceFor(cov, serial),
			Observations:  cov.Observations,
			SerialMinSeen: cov.SerialMinSeen, SerialMaxSeen: cov.SerialMaxSeen,
			RangeID: cov.ID,
		}
	}
	return out, nil
}

// ResolveByBuildKey returns every confirmed range per field — the build-key view.
func (e *ForkEngine) ResolveByBuildKey(buildKey string, fieldKeys []string) (map[string][]RangeView, error) {
	out := make(map[string][]RangeView, len(fieldKeys))
	for _, fk := range fieldKeys {
		ranges, err := e.Store.RangesFor(buildKey, fk)
		if err != nil {
			return nil, err
		}
		sort.Slice(ranges, func(i, j int) bool { return ranges[i].SerialStart < ranges[j].SerialStart })
		views := make([]RangeView, 0, len(ranges))
		for _, r := range ranges {
			views = append(views, RangeView{
				SerialStart: r.SerialStart, SerialEnd: r.SerialEnd, Value: r.Value,
				Origin: r.Origin, Source: r.Source, Observations: r.Observations,
				BoundaryExact: r.BoundaryExact,
			})
		}
		if len(views) > 0 {
			out[fk] = views
		}
	}
	return out, nil
}

func confidenceFor(r *models.FieldRange, serial int64) ConfidenceTier {
	if r.Origin == "manual" {
		return ConfManual
	}
	if serial < r.SerialMinSeen || serial > r.SerialMaxSeen {
		return ConfExtrapolated
	}
	if r.Observations >= 2 {
		return ConfObserved
	}
	return ConfAssumed
}

// ─── helpers ──────────────────────────────────────────────────────────────────

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
	max := start
	if end != nil {
		max = *end
	}
	return &models.FieldRange{
		BuildKey: src.BuildKey, FieldKey: src.FieldKey,
		SerialStart: start, SerialEnd: end,
		Value: src.Value, ValueNorm: src.ValueNorm, Origin: src.Origin, Source: src.Source,
		Observations: src.Observations, SerialMinSeen: start, SerialMaxSeen: max,
		BoundaryExact: false, LastActor: src.LastActor,
		CreatedAt: now, UpdatedAt: now,
	}
}

// ─── edit-path integration helpers ─────────────────────────────────────────────

// SerialFromVIN extracts the build number (positions 12–17) as an integer from a full
// 17-character VIN whose serial is numeric. Returns (0,false) for build-key lookups
// (len != 17) or non-numeric serials (treat that build key as un-forkable).
func SerialFromVIN(vin string) (int64, bool) {
	vin = strings.TrimSpace(strings.ToUpper(vin))
	if len(vin) != 17 {
		return 0, false
	}
	n, err := strconv.ParseInt(vin[11:17], 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// IsForkFieldKey reports whether a field is a registered fork field for this make/model.
// Lets callers (e.g. the edit handler) decide a field's tier without the full engine.
func IsForkFieldKey(db *gorm.DB, make_, model, fieldKey string) bool {
	ok, err := NewGormForkStore(db).IsForkField(make_, model, fieldKey)
	return err == nil && ok
}

// PendingPointsFor returns the un-absorbed pending sightings per field for a build key
// (the single verified observations still waiting for a second to form a range).
func PendingPointsFor(db *gorm.DB, buildKey string) (map[string][]int64, error) {
	var pts []models.FieldPoint
	if err := db.Where("build_key = ?", buildKey).
		Order("field_key ASC, serial ASC").Find(&pts).Error; err != nil {
		return nil, err
	}
	out := make(map[string][]int64)
	for _, p := range pts {
		out[p.FieldKey] = append(out[p.FieldKey], p.Serial)
	}
	return out, nil
}

// ForkFieldKeysFor returns the enabled fork-field keys that apply to a make/model
// (global + make-scoped + make/model-scoped).
func ForkFieldKeysFor(db *gorm.DB, make_, model string) ([]string, error) {
	var keys []string
	err := db.Model(&models.ForkField{}).
		Where("enabled = ?", true).
		Where("(scope_make = '' OR scope_make = ?)", make_).
		Where("(scope_model = '' OR scope_model = ?)", model).
		Distinct().
		Pluck("key", &keys).Error
	return keys, err
}

// FeedForkField records a VERIFIED fork-field value into the range engine. A VIN sighting
// (serial set) goes through RecordPoint (two-point rule); a build-key-wide value (serial
// nil) is written as an authoritative manual base range. No-op + (false,nil) if the field
// is not a registered fork field. Always call with verified/authoritative data only.
func FeedForkField(db *gorm.DB, buildKey, make_, model, fieldKey, value string,
	serial *int64, source string, actorID *uint) (bool, error) {

	store := NewGormForkStore(db)
	isFork, err := store.IsForkField(make_, model, fieldKey)
	if err != nil || !isFork {
		return false, err
	}
	engine := NewForkEngine(store)
	if serial != nil {
		_, err = engine.RecordPoint(PointInput{
			BuildKey: buildKey, Serial: *serial, FieldKey: fieldKey, Value: value,
			Source: source, Verified: true, ActorID: actorID, Make: make_, Model: model,
		})
	} else {
		_, err = engine.RecordRange(RangeInput{
			BuildKey: buildKey, SerialStart: 0, SerialEnd: nil, FieldKey: fieldKey,
			Value: value, Source: source, Verified: true, ActorID: actorID,
			Make: make_, Model: model,
		})
	}
	return true, err
}

// ─── note applicability ────────────────────────────────────────────────────────

// Note scope values describe whether a note applies to the VIN being viewed.
const (
	NoteScopeApplies = "applies" // confidently applies to this VIN
	NoteScopeWarn    = "warn"    // shown, but flagged (different range / unknown / unscoped)
)

// ForkBoundaries returns the distinct range start serials across ALL fork fields for a
// build key — i.e. every serial at which some build-number field is known to change.
func ForkBoundaries(db *gorm.DB, buildKey string) ([]int64, error) {
	var starts []int64
	err := db.Model(&models.FieldRange{}).
		Where("build_key = ?", buildKey).
		Distinct().
		Order("serial_start ASC").
		Pluck("serial_start", &starts).Error
	return starts, err
}

// NoteScope decides whether a note applies to the VIN being viewed. A note applies only
// when both serials are known, we have range data, and NO fork boundary lies between the
// note's serial and the viewed serial. Otherwise it is shown with a warning.
func NoteScope(boundaries []int64, noteSerial, viewSerial *int64) string {
	if noteSerial == nil || viewSerial == nil {
		return NoteScopeWarn // unscoped, or viewing a build key with no specific serial
	}
	if len(boundaries) == 0 {
		return NoteScopeWarn // no range data — we don't know the range
	}
	if *noteSerial == *viewSerial {
		return NoteScopeApplies
	}
	lo, hi := *noteSerial, *viewSerial
	if lo > hi {
		lo, hi = hi, lo
	}
	for _, b := range boundaries {
		if b > lo && b <= hi {
			return NoteScopeWarn // a build-number change lies between the two VINs
		}
	}
	return NoteScopeApplies
}

// ─── seeding ────────────────────────────────────────────────────────────────

// DefaultForkFields are the build-number-tier fields tracked out of the box (global scope).
var DefaultForkFields = []string{
	"brake_code",
	"front_rotor_size",
	"rear_rotor_size",
	"front_suspension",
	"rear_suspension",
	"steering",
}

// SeedDefaultForkFields inserts the default fork-field registry entries if missing.
// Idempotent — safe to call on every boot.
func SeedDefaultForkFields(db *gorm.DB) error {
	for _, key := range DefaultForkFields {
		var count int64
		if err := db.Model(&models.ForkField{}).
			Where("key = ? AND scope_make = '' AND scope_model = ''", key).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&models.ForkField{
			Key: key, ScopeMake: "", ScopeModel: "",
			ValueType: "string", Enabled: true,
			Description: "default build-number fork field",
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// ─── gorm-backed store ─────────────────────────────────────────────────────────

type gormForkStore struct{ db *gorm.DB }

// NewGormForkStore returns a ForkStore backed by the database.
func NewGormForkStore(db *gorm.DB) ForkStore { return &gormForkStore{db: db} }

func (s *gormForkStore) IsForkField(make_, model, fieldKey string) (bool, error) {
	var count int64
	err := s.db.Model(&models.ForkField{}).
		Where("key = ? AND enabled = ?", fieldKey, true).
		Where("(scope_make = '' OR scope_make = ?)", make_).
		Where("(scope_model = '' OR scope_model = ?)", model).
		Count(&count).Error
	return count > 0, err
}

func (s *gormForkStore) RangesFor(buildKey, fieldKey string) ([]models.FieldRange, error) {
	var rs []models.FieldRange
	err := s.db.Where("build_key = ? AND field_key = ?", buildKey, fieldKey).
		Order("serial_start ASC").Find(&rs).Error
	return rs, err
}

func (s *gormForkStore) PointsFor(buildKey, fieldKey string) ([]models.FieldPoint, error) {
	var ps []models.FieldPoint
	err := s.db.Where("build_key = ? AND field_key = ?", buildKey, fieldKey).
		Order("serial ASC").Find(&ps).Error
	return ps, err
}

func (s *gormForkStore) SaveRange(r *models.FieldRange) error { return s.db.Save(r).Error }
func (s *gormForkStore) DeleteRange(id uint) error {
	return s.db.Delete(&models.FieldRange{}, id).Error
}
func (s *gormForkStore) SavePoint(p *models.FieldPoint) error { return s.db.Save(p).Error }
func (s *gormForkStore) DeletePoints(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.Delete(&models.FieldPoint{}, ids).Error
}
