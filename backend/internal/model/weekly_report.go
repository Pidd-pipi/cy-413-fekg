package model

import "time"

// WeeklyReport 是 14 天情绪周报的固化快照。
// (user_id, end_date) 唯一索引保证同一账号同一结束日只保留一份，
// 快照生成后不再随情绪记录或日记的修改/删除而回写。
type WeeklyReport struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"not null;uniqueIndex:uniq_weekly_report_user_end,priority:1" json:"user_id"`
	StartDate   time.Time  `gorm:"type:date;not null" json:"start_date"`
	EndDate     time.Time  `gorm:"type:date;not null;uniqueIndex:uniq_weekly_report_user_end,priority:2" json:"end_date"`
	AverageMood float64    `gorm:"not null;default:0" json:"average_mood"`
	LowestDate  *time.Time `gorm:"type:date" json:"lowest_date,omitempty"`
	LowestLevel int        `gorm:"not null;default:0" json:"lowest_level"`
	ValidDays   int        `gorm:"not null;default:0" json:"valid_days"`
	Days        string     `gorm:"type:text;not null" json:"-"`
	TagCounts   string     `gorm:"type:text;not null" json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
}
