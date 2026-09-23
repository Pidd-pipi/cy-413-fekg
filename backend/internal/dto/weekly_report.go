package dto

// WeeklyReportGenerateRequest 用于指定周报结束日（自然日），缺省为今天。
type WeeklyReportGenerateRequest struct {
	EndDate string `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
}

// WeeklyReportDay 是快照中每个有效自然日的固化条目。
type WeeklyReportDay struct {
	Date         string `json:"date"`
	MoodLevel    int    `json:"mood_level"`
	JournalTitle string `json:"journal_title"`
	JournalMood  int    `json:"journal_mood"`
	HasJournal   bool   `json:"has_journal"`
}

// WeeklyReportTagCount 是标签频次统计条目，按频次降序输出。
type WeeklyReportTagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// WeeklyReportData 是写入 weekly_reports.snapshot 的固化内容。
type WeeklyReportData struct {
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	Average     float64                `json:"average"`
	LowestDay   *WeeklyReportDay       `json:"lowest_day"`
	TagCounts   []WeeklyReportTagCount `json:"tag_counts"`
	ValidDays   int                    `json:"valid_days"`
	Days        []WeeklyReportDay      `json:"days"`
	GeneratedAt string                 `json:"generated_at"`
}

// WeeklyReportResponse 是周报接口对外的统一响应。
type WeeklyReportResponse struct {
	ID        uint             `json:"id"`
	StartDate string           `json:"start_date"`
	EndDate   string           `json:"end_date"`
	Reused    bool             `json:"reused"`
	CreatedAt string           `json:"created_at"`
	Report    WeeklyReportData `json:"report"`
}
