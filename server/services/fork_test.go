package services

import (
	"sort"
	"testing"

	"main/models"
)

// ─── in-memory ForkStore for testing the algorithm without a DB ────────────────

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
func (m *memForkStore) SavePoint(p *models.FieldPoint) error {
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

// ─── helpers ───────────────────────────────────────────────────────────────────

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

// ─── tests ─────────────────────────────────────────────────────────────────────

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

func derefOr(p *int64) int64 {
	if p == nil {
		return -1
	}
	return *p
}
