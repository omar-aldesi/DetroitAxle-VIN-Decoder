package services

import (
	"sort"

	"main/models"
)

// ConfidenceTier describes how trustworthy a resolved value is.
type ConfidenceTier string

const (
	ConfObserved     ConfidenceTier = "observed"     // within evidence of a confirmed range
	ConfAssumed      ConfidenceTier = "assumed"      // within range, thin evidence
	ConfExtrapolated ConfidenceTier = "extrapolated" // beyond observed evidence
	ConfManual       ConfidenceTier = "manual"       // human-entered
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
