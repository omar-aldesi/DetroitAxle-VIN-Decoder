package services

import (
	"math"
	"time"

	"main/models"
)

const openEnd = int64(math.MaxInt64) // sentinel for an open-ended range during comparisons

// ForkOutcome reports what a write did.
type ForkOutcome string

const (
	OutcomeIgnored      ForkOutcome = "ignored"       
	OutcomeNotForkField ForkOutcome = "not_fork_field" // caller should treat as a build-key field
	OutcomePending      ForkOutcome = "pending"       
	OutcomeReinforced   ForkOutcome = "reinforced"
	OutcomeRangeCreated ForkOutcome = "range_created"
	OutcomeForked       ForkOutcome = "forked"
	OutcomeManualRange  ForkOutcome = "manual_range"
	OutcomeBaseRange    ForkOutcome = "base_range" // gap-filling build-key-wide base value
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
