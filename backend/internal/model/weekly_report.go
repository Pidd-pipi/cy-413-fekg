package model

import "time"

// WeeklyReport 是用户某一结束日对应的 14 天情绪周报快照。
// 同一 (user_id, end_date) 唯一；生成后 Snapshot 固化，不再随情绪/日记的改删回写。
type WeeklyReport struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_weekly_report_user_end;not null" json:"user_id"`
	EndDate   time.Time `gorm:"type:date;uniqueIndex:uk_weekly_report_user_end;not null" json:"end_date"`
	StartDate time.Time `gorm:"type:date;not null" json:"start_date"`
	Snapshot  string    `gorm:"type:text;not null" json:"snapshot"`
	CreatedAt time.Time `json:"created_at"`
}
