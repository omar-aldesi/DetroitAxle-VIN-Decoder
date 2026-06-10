package services

import (
	"sort"
	"testing"

	"main/models"
)

type memForkStore struct {
	fields map[string]bool
	ranges map[uint]*models.FieldRange
	points map[uint]*models.FieldPoint
	nextR  uint
	nextP  uint
}

func newMemStore(fields ...string) *memForkStore {
	m := &memForkStore{
		fields: map[string]bool{},
		ranges: map[uint]*models.FieldRange{},
		points: map[uint]*models.FieldPoint{},
	}
	for _, f := range fields {
		m.fields[f] = true
	}
	return m
}

func (m *memForkStore) IsForkField(_, _, key string) (bool, error) { return m.fields[key], nil }

func (m *memForkStore) RangesFor(bk, fk string) ([]models.FieldRange, error) {
	var out []models.FieldRange
	for _, r := range m.ranges {
		if r.BuildKey == bk && r.FieldKey == fk {
			out = append(out, *r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SerialStart < out[j].SerialStart })
	return out, nil
}

func (m *memForkStore) PointsFor(bk, fk string) ([]models.FieldPoint, error) {
	var out []models.FieldPoint
	for _, p := range m.points {
		if p.BuildKey == bk && p.FieldKey == fk {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Serial < out[j].Serial })
	return out, nil
}

func (m *memForkStore) SaveRange(r *models.FieldRange) error {
	if r.ID == 0 {
		m.nextR++
		r.ID = m.nextR
	}
	cp := *r
	m.ranges[r.ID] = &cp
	return nil
}
func (m *memForkStore) DeleteRange(id uint) error { delete(m.ranges, id); return nil }
// SavePoint mirrors the gorm store: upsert on (build_key, field_key, serial).
func (m *memForkStore) SavePoint(p *models.FieldPoint) error {
	if p.ID == 0 {
		for id, q := range m.points {
			if q.BuildKey == p.BuildKey && q.FieldKey == p.FieldKey && q.Serial == p.Serial {
				p.ID = id
				break
			}
		}
	}
	if p.ID == 0 {
		m.nextP++
		p.ID = m.nextP
	}
	cp := *p
	m.points[p.ID] = &cp
	return nil
}
func (m *memForkStore) DeletePoints(ids []uint) error {
	for _, id := range ids {
		delete(m.points, id)
	}
	return nil
}

const bk = "1G1AK55F77"

func ptr(v int64) *int64 { return &v }

// rp records a verified VIN sighting for brake_code and returns the outcome.
func rp(t *testing.T, e *ForkEngine, serial int64, value string) ForkOutcome {
	t.Helper()
	o, err := e.RecordPoint(PointInput{
		BuildKey: bk, Serial: serial, FieldKey: "brake_code", Value: value,
		Source: "test", Verified: true,
	})
	if err != nil {
		t.Fatalf("RecordPoint(%d,%q): %v", serial, value, err)
	}
	return o
}

func resolveVal(t *testing.T, e *ForkEngine, serial int64) (string, bool) {
	t.Helper()
	m, err := e.Resolve(bk, serial, []string{"brake_code"})
	if err != nil {
		t.Fatalf("Resolve(%d): %v", serial, err)
	}
	r, ok := m["brake_code"]
	return r.Value, ok
}

func TestFork_TwoPointRule(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))

	if got := rp(t, e, 1, "JP9"); got != OutcomePending {
		t.Fatalf("first sighting = %s, want pending (rule #1)", got)
	}
	if _, ok := resolveVal(t, e, 1); ok {
		t.Fatal("a single sighting must not resolve to a value yet")
	}

	if got := rp(t, e, 100, "JP9"); got != OutcomeRangeCreated {
		t.Fatalf("second agreeing sighting = %s, want range_created", got)
	}
	if v, ok := resolveVal(t, e, 50); !ok || v != "JP9" {
		t.Fatalf("serial 50 = (%q,%v), want JP9 (interpolated)", v, ok)
	}
	if _, ok := resolveVal(t, e, 150); ok {
		t.Fatal("serial 150 is beyond the observed range; should be unknown")
	}
}

func TestFork_Reinforce(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))
	rp(t, e, 1, "JP9")
	rp(t, e, 100, "JP9")
	if got := rp(t, e, 50, "JP9"); got != OutcomeReinforced {
		t.Fatalf("matching sighting inside range = %s, want reinforced", got)
	}
	m, _ := e.Resolve(bk, 50, []string{"brake_code"})
	if m["brake_code"].Observations != 3 {
		t.Fatalf("observations = %d, want 3", m["brake_code"].Observations)
	}
	if m["brake_code"].Confidence != ConfObserved {
		t.Fatalf("confidence = %s, want observed", m["brake_code"].Confidence)
	}
}

// The user's headline walkthrough: 001/100 = JP9, then a JP6 run, then JP9 returns at 352.
func TestFork_Walkthrough_001_100_101_352(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))

	rp(t, e, 1, "JP9")
	rp(t, e, 100, "JP9") // → [1..100] JP9

	if got := rp(t, e, 101, "JP6"); got != OutcomePending {
		t.Fatalf("101 = %s, want pending (needs a 2nd JP6)", got)
	}
	rp(t, e, 200, "JP6") // → [101..200] JP6

	if got := rp(t, e, 352, "JP9"); got != OutcomePending {
		t.Fatalf("352 = %s, want pending (needs a 2nd JP9)", got)
	}
	rp(t, e, 400, "JP9") // → [352..400] JP9 (new fork, never went backwards)

	views, _ := e.ResolveByBuildKey(bk, []string{"brake_code"})
	got := views["brake_code"]
	want := []struct {
		start int64
		end   int64
		val   string
	}{{1, 100, "JP9"}, {101, 200, "JP6"}, {352, 400, "JP9"}}
	if len(got) != len(want) {
		t.Fatalf("got %d ranges, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].SerialStart != w.start || got[i].SerialEnd == nil || *got[i].SerialEnd != w.end || got[i].Value != w.val {
			t.Errorf("range %d = [%d..%v]=%s, want [%d..%d]=%s",
				i, got[i].SerialStart, derefOr(got[i].SerialEnd), got[i].Value, w.start, w.end, w.val)
		}
	}
}

// A contradiction inside an established range is held aside until a 2nd agreeing exception,
// then carved as an inner range (3-way split).
func TestFork_CarveInsideRange(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))
	rp(t, e, 1, "JP9")
	rp(t, e, 200, "JP9") // → [1..200] JP9

	if got := rp(t, e, 50, "JP6"); got != OutcomePending {
		t.Fatalf("first exception 50 = %s, want pending", got)
	}
	if v, _ := resolveVal(t, e, 50); v != "JP9" {
		t.Fatalf("serial 50 before carve = %q, want JP9 (exception not yet applied)", v)
	}

	if got := rp(t, e, 60, "JP6"); got != OutcomeForked {
		t.Fatalf("second exception 60 = %s, want forked", got)
	}

	checks := map[int64]string{30: "JP9", 50: "JP6", 55: "JP6", 60: "JP6", 100: "JP9", 200: "JP9"}
	for serial, want := range checks {
		if v, ok := resolveVal(t, e, serial); !ok || v != want {
			t.Errorf("serial %d = (%q,%v), want %s", serial, v, ok, want)
		}
	}
}

func TestFork_ManualRange_WinsAndTrims(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))
	rp(t, e, 1, "JP9")
	rp(t, e, 200, "JP9") // → [1..200] JP9

	o, err := e.RecordRange(RangeInput{
		BuildKey: bk, SerialStart: 50, SerialEnd: ptr(100), FieldKey: "brake_code",
		Value: "JP6", Source: "dnr", Verified: true,
	})
	if err != nil || o != OutcomeManualRange {
		t.Fatalf("RecordRange = (%s,%v), want manual_range", o, err)
	}

	// Expect [1..49] JP9, [50..100] JP6 (manual), [101..200] JP9.
	checks := map[int64]string{25: "JP9", 49: "JP9", 50: "JP6", 100: "JP6", 101: "JP9", 200: "JP9"}
	for serial, want := range checks {
		if v, ok := resolveVal(t, e, serial); !ok || v != want {
			t.Errorf("serial %d = (%q,%v), want %s", serial, v, ok, want)
		}
	}
	m, _ := e.Resolve(bk, 75, []string{"brake_code"})
	if m["brake_code"].Origin != "manual" || m["brake_code"].Confidence != ConfManual {
		t.Errorf("manual span should resolve origin=manual/confidence=manual, got %+v", m["brake_code"])
	}
}

func TestFork_UnverifiedDoesNothing(t *testing.T) {
	store := newMemStore("brake_code")
	e := NewForkEngine(store)
	o, _ := e.RecordPoint(PointInput{BuildKey: bk, Serial: 1, FieldKey: "brake_code", Value: "JP9", Verified: false})
	if o != OutcomeIgnored {
		t.Fatalf("unverified = %s, want ignored", o)
	}
	if len(store.points) != 0 || len(store.ranges) != 0 {
		t.Fatal("unverified data must not be stored anywhere")
	}
}

func TestFork_NotForkField(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code")) // "color" not registered
	o, _ := e.RecordPoint(PointInput{BuildKey: bk, Serial: 1, FieldKey: "color", Value: "Red", Verified: true})
	if o != OutcomeNotForkField {
		t.Fatalf("unregistered field = %s, want not_fork_field", o)
	}
}

func TestFork_NormalizationNoPhantomFork(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))
	rp(t, e, 1, "JP9")
	if got := rp(t, e, 100, "  jp9 "); got != OutcomeRangeCreated {
		t.Fatalf("normalized-equal values should form ONE range, got %s", got)
	}
	views, _ := e.ResolveByBuildKey(bk, []string{"brake_code"})
	if len(views["brake_code"]) != 1 {
		t.Fatalf("expected 1 range, got %d (phantom fork from casing/whitespace)", len(views["brake_code"]))
	}
}

func TestSerialFromVIN(t *testing.T) {
	cases := []struct {
		vin  string
		want int64
		ok   bool
	}{
		{"1G1AK55F177000001", 1, true},
		{"1G1AK55F177000100", 100, true},
		{"1G1AK55F77", 0, false},        // build key, not a full VIN
		{"1G1AK55F177ABC100", 0, false}, // non-numeric serial → un-forkable
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := SerialFromVIN(c.vin)
		if got != c.want || ok != c.ok {
			t.Errorf("SerialFromVIN(%q) = (%d,%v), want (%d,%v)", c.vin, got, ok, c.want, c.ok)
		}
	}
}

func TestNoteScope(t *testing.T) {
	boundaries := []int64{1, 101, 352} // change points from the walkthrough
	cases := []struct {
		name       string
		boundaries []int64
		note, view *int64
		want       string
	}{
		{"same segment applies", boundaries, ptr(50), ptr(75), NoteScopeApplies},
		{"boundary between warns", boundaries, ptr(50), ptr(150), NoteScopeWarn},
		{"same serial applies", boundaries, ptr(120), ptr(120), NoteScopeApplies},
		{"same serial applies even without range data", nil, ptr(120), ptr(120), NoteScopeApplies},
		{"unscoped note warns", boundaries, nil, ptr(50), NoteScopeWarn},
		{"no view serial warns", boundaries, ptr(50), nil, NoteScopeWarn},
		{"no range data warns", nil, ptr(50), ptr(75), NoteScopeWarn},
		{"order-independent", boundaries, ptr(150), ptr(50), NoteScopeWarn},
	}
	for _, c := range cases {
		if got := NoteScope(c.boundaries, c.note, c.view); got != c.want {
			t.Errorf("%s: NoteScope = %q, want %q", c.name, got, c.want)
		}
	}
}

func derefOr(p *int64) int64 {
	if p == nil {
		return -1
	}
	return *p
}

// Re-verifying the same VIN with a corrected value must replace the old sighting
// (upsert), not error on the unique index, and the corrected value must win.
func TestFork_CorrectedSightingSameSerial(t *testing.T) {
	store := newMemStore("brake_code")
	e := NewForkEngine(store)

	rp(t, e, 1, "JP9")
	if got := rp(t, e, 1, "JP6"); got != OutcomePending {
		t.Fatalf("corrected sighting = %s, want pending", got)
	}
	if len(store.points) != 1 {
		t.Fatalf("expected 1 point after correction, got %d", len(store.points))
	}

	rp(t, e, 100, "JP6")
	if v, ok := resolveVal(t, e, 50); !ok || v != "JP6" {
		t.Fatalf("serial 50 = (%q,%v), want JP6 (corrected value wins)", v, ok)
	}
}

// Agreeing points in different gaps (an existing range between them) must each wait for
// their own second sighting — and form per-gap ranges, never one straddling range.
func TestFork_GapGrouping(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))

	rp(t, e, 200, "JP6")
	rp(t, e, 300, "JP6") // → [200..300] JP6

	if got := rp(t, e, 100, "JP9"); got != OutcomePending {
		t.Fatalf("100 = %s, want pending", got)
	}
	if got := rp(t, e, 500, "JP9"); got != OutcomePending {
		t.Fatalf("500 = %s, want pending (different gap than 100 — range between)", got)
	}

	if got := rp(t, e, 510, "JP9"); got != OutcomeRangeCreated {
		t.Fatalf("510 = %s, want range_created (second JP9 in the upper gap)", got)
	}
	if v, ok := resolveVal(t, e, 505); !ok || v != "JP9" {
		t.Fatalf("serial 505 = (%q,%v), want JP9", v, ok)
	}
	if v, _ := resolveVal(t, e, 250); v != "JP6" {
		t.Fatalf("serial 250 = %q, want JP6 (existing range untouched)", v)
	}
	if _, ok := resolveVal(t, e, 100); ok {
		t.Fatal("serial 100 should still be unknown (lower gap has only one sighting)")
	}

	if got := rp(t, e, 110, "JP9"); got != OutcomeRangeCreated {
		t.Fatalf("110 = %s, want range_created (second JP9 in the lower gap)", got)
	}
	if v, ok := resolveVal(t, e, 105); !ok || v != "JP9" {
		t.Fatalf("serial 105 = (%q,%v), want JP9", v, ok)
	}
}

// A build-key-wide base value fills only the gaps — it must never erase observed ranges.
func TestFork_BaseRangeFillsGapsOnly(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))

	rp(t, e, 100, "JP6")
	rp(t, e, 200, "JP6") // → [100..200] JP6

	o, err := e.RecordBaseRange(RangeInput{
		BuildKey: bk, SerialStart: 0, FieldKey: "brake_code",
		Value: "JP9", Source: "verified", Verified: true,
	})
	if err != nil || o != OutcomeBaseRange {
		t.Fatalf("RecordBaseRange = (%s,%v), want base_range", o, err)
	}

	checks := map[int64]string{0: "JP9", 50: "JP9", 99: "JP9", 100: "JP6", 200: "JP6", 201: "JP9", 99999: "JP9"}
	for serial, want := range checks {
		if v, ok := resolveVal(t, e, serial); !ok || v != want {
			t.Errorf("serial %d = (%q,%v), want %s", serial, v, ok, want)
		}
	}

	// Re-running with another value is a no-op: everything is already covered.
	o, err = e.RecordBaseRange(RangeInput{
		BuildKey: bk, SerialStart: 0, FieldKey: "brake_code",
		Value: "JP4", Source: "legacy", Verified: true,
	})
	if err != nil || o != OutcomeBaseRange {
		t.Fatalf("re-run RecordBaseRange = (%s,%v), want base_range", o, err)
	}
	if v, _ := resolveVal(t, e, 50); v != "JP9" {
		t.Fatalf("serial 50 = %q, want JP9 (re-run must not overwrite covered ranges)", v)
	}
	if v, _ := resolveVal(t, e, 150); v != "JP6" {
		t.Fatalf("serial 150 = %q, want JP6 (re-run must not overwrite observed range)", v)
	}
}

// A pending contradiction later covered by an agreeing range must be absorbed,
// not linger as a phantom pending sighting.
func TestFork_StalePendingAbsorbed(t *testing.T) {
	store := newMemStore("brake_code")
	e := NewForkEngine(store)

	rp(t, e, 1, "JP9")
	rp(t, e, 200, "JP9") // → [1..200] JP9

	if got := rp(t, e, 50, "JP6"); got != OutcomePending {
		t.Fatalf("exception 50 = %s, want pending", got)
	}
	if len(store.points) != 1 {
		t.Fatalf("expected 1 pending point, got %d", len(store.points))
	}

	// DNR confirms the exception span manually with the same value.
	if o, err := e.RecordRange(RangeInput{
		BuildKey: bk, SerialStart: 40, SerialEnd: ptr(60), FieldKey: "brake_code",
		Value: "JP6", Source: "dnr", Verified: true,
	}); err != nil || o != OutcomeManualRange {
		t.Fatalf("RecordRange = (%s,%v), want manual_range", o, err)
	}

	if len(store.points) != 0 {
		t.Fatalf("pending point should be absorbed by the agreeing manual range, %d left", len(store.points))
	}
}

// Remnants of a trim/carve keep only the evidence inside them; a remnant with no real
// evidence must not resolve as "observed".
func TestFork_RemnantEvidenceHonest(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))

	rp(t, e, 1, "JP9")
	rp(t, e, 100, "JP9") // → [1..100] JP9, evidence at 1 and 100

	// Manual range over the top end: right remnant would be empty, left remnant [1..49]
	// keeps the evidence at serial 1 only.
	if o, err := e.RecordRange(RangeInput{
		BuildKey: bk, SerialStart: 50, SerialEnd: ptr(100), FieldKey: "brake_code",
		Value: "JP6", Source: "dnr", Verified: true,
	}); err != nil || o != OutcomeManualRange {
		t.Fatalf("RecordRange = (%s,%v), want manual_range", o, err)
	}

	m, _ := e.Resolve(bk, 25, []string{"brake_code"})
	r := m["brake_code"]
	if r.Value != "JP9" {
		t.Fatalf("serial 25 = %q, want JP9", r.Value)
	}
	if r.Confidence == ConfObserved {
		t.Fatalf("serial 25 confidence = %s; remnant evidence is only serial 1 — must not be observed", r.Confidence)
	}
	if r.SerialMaxSeen > 49 {
		t.Fatalf("left remnant SerialMaxSeen = %d, want <= 49 (evidence at 100 was cut away)", r.SerialMaxSeen)
	}
}

// A carve absorbing 3 agreeing exceptions must record 3 observations, not a hardcoded 2.
func TestFork_CarveObservationCount(t *testing.T) {
	e := NewForkEngine(newMemStore("brake_code"))

	rp(t, e, 1, "JP9")
	rp(t, e, 300, "JP9") // → [1..300] JP9

	// Two exceptions carve; a third inside reinforces the carved range.
	rp(t, e, 50, "JP6")
	if got := rp(t, e, 70, "JP6"); got != OutcomeForked {
		t.Fatalf("second exception = %s, want forked", got)
	}
	if got := rp(t, e, 60, "JP6"); got != OutcomeReinforced {
		t.Fatalf("third matching sighting = %s, want reinforced", got)
	}

	m, _ := e.Resolve(bk, 60, []string{"brake_code"})
	if m["brake_code"].Observations != 3 {
		t.Fatalf("observations = %d, want 3 (2 carved + 1 reinforced)", m["brake_code"].Observations)
	}
}
