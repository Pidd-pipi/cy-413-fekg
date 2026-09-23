package dto

// WeeklyReportDay 是周报中某个自然日的固化快照条目：
// 情绪取当日（record_date, id）最后一条，日记取当日（created_at, id）最后一篇。
type WeeklyReportDay struct {
	Date         string   `json:"date"`
	MoodID       uint     `json:"mood_id"`
	MoodLevel    int      `json:"mood_level"`
	MoodTags     []string `json:"mood_tags"`
	JournalID    uint     `json:"journal_id,omitempty"`
	JournalTitle string   `json:"journal_title,omitempty"`
	JournalMood  int      `json:"journal_mood,omitempty"`
	HasJournal   bool     `json:"has_journal"`
}

// WeeklyReportTag 是标签频次条目，按 count desc、tag asc 排序。
type WeeklyReportTag struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// WeeklyReportData 是周报的完整视图，由 model.WeeklyReport 反序列化组装。
type WeeklyReportData struct {
	ID           uint              `json:"id"`
	UserID       uint              `json:"user_id"`
	StartDate    string            `json:"start_date"`
	EndDate      string            `json:"end_date"`
	AverageMood  float64           `json:"average_mood"`
	LowestDate   string            `json:"lowest_date,omitempty"`
	LowestLevel  int               `json:"lowest_level"`
	ValidDays    int               `json:"valid_days"`
	TagCounts    []WeeklyReportTag `json:"tag_counts"`
	Days         []WeeklyReportDay `json:"days"`
	AlreadyExist bool              `json:"already_exists"`
	CreatedAt    string            `json:"created_at"`
}
