package repository

import (
	"errors"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"gorm.io/gorm"
)

type WeeklyReportRepository interface {
	Create(*model.WeeklyReport) error
	ByUserAndEndDate(uint, string) (*model.WeeklyReport, error)
}
type weeklyReportRepository struct{ db *gorm.DB }

func NewWeeklyReportRepository(db *gorm.DB) WeeklyReportRepository {
	return &weeklyReportRepository{db}
}
func (r *weeklyReportRepository) Create(v *model.WeeklyReport) error { return r.db.Create(v).Error }

// ByUserAndEndDate 按 (user_id, end_date) 读取已固化的周报快照；endDate 形如 2006-01-02。
func (r *weeklyReportRepository) ByUserAndEndDate(uid uint, endDate string) (*model.WeeklyReport, error) {
	var v model.WeeklyReport
	e := r.db.Where("user_id = ? AND end_date = ?", uid, endDate).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}
