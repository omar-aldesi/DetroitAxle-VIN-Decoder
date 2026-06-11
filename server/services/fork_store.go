package services

import (
	"main/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

// SavePoint upserts on (build_key, field_key, serial): re-verifying the same VIN with a
// corrected value replaces the old sighting instead of failing the unique index.
func (s *gormForkStore) SavePoint(p *models.FieldPoint) error {
	if p.ID != 0 {
		return s.db.Save(p).Error
	}
	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "build_key"}, {Name: "field_key"}, {Name: "serial"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value", "value_norm", "source", "actor", "created_at",
		}),
	}).Create(p).Error
}
func (s *gormForkStore) DeletePoints(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.Delete(&models.FieldPoint{}, ids).Error
}
