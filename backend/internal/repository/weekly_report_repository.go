package repository

import (
	"errors"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type WeeklyReportRepository interface {
	// CreateIgnoreConflict 依赖 (user_id, end_date) 唯一索引做幂等写入：
	// 并发生成时只有一份落库，冲突方拿到 created=false 后回读已有快照。
	CreateIgnoreConflict(*model.WeeklyReport) (bool, error)
	ByEndDate(uid uint, endDate time.Time) (*model.WeeklyReport, error)
	ByID(id, uid uint) (*model.WeeklyReport, error)
	Latest(uid uint) (*model.WeeklyReport, error)
}

type weeklyReportRepository struct{ db *gorm.DB }

func NewWeeklyReportRepository(db *gorm.DB) WeeklyReportRepository {
	return &weeklyReportRepository{db}
}

func (r *weeklyReportRepository) CreateIgnoreConflict(v *model.WeeklyReport) (bool, error) {
	res := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "end_date"}},
		DoNothing: true,
	}).Create(v)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *weeklyReportRepository) ByEndDate(uid uint, endDate time.Time) (*model.WeeklyReport, error) {
	var v model.WeeklyReport
	e := r.db.Where("user_id = ? AND end_date = ?::date", uid, endDate.Format("2006-01-02")).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *weeklyReportRepository) ByID(id, uid uint) (*model.WeeklyReport, error) {
	var v model.WeeklyReport
	e := r.db.Where("id = ? AND user_id = ?", id, uid).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *weeklyReportRepository) Latest(uid uint) (*model.WeeklyReport, error) {
	var v model.WeeklyReport
	e := r.db.Where("user_id = ?", uid).Order("end_date desc, id desc").First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}
