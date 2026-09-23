package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/util"
)

// WeeklyReportDays 周报覆盖的自然日数量（含结束日）。
const WeeklyReportDays = 14

const dayLayout = "2006-01-02"

type WeeklyReportService struct {
	repo     repository.WeeklyReportRepository
	moods    repository.MoodRepository
	journals repository.JournalRepository
	logger   *slog.Logger
}

func NewWeeklyReportService(r repository.WeeklyReportRepository, m repository.MoodRepository, j repository.JournalRepository, l *slog.Logger) *WeeklyReportService {
	return &WeeklyReportService{repo: r, moods: m, journals: j, logger: l}
}

// Generate 生成（或取回）指定结束日的 14 天周报。
// 同一账号同一结束日已有快照时原样返回（reused=true），绝不覆盖；
// 并发生成依赖唯一索引保证只保留一份。
func (s *WeeklyReportService) Generate(uid uint, endDate string) (*dto.WeeklyReportResponse, error) {
	end, e := resolveEndDate(endDate)
	if e != nil {
		return nil, util.NewAppError(constants.CodeValidation, "WeeklyReport[end_date] generate failed: invalid date", e)
	}
	if existing, e := s.repo.ByEndDate(uid, end); e == nil {
		return s.toResponse(existing, true)
	} else if e != repository.ErrNotFound {
		return nil, fmt.Errorf("WeeklyReport[user_id] fetch failed: %w", e)
	}

	start := end.AddDate(0, 0, -(WeeklyReportDays - 1))
	rangeEnd := end.AddDate(0, 0, 1)

	moods, e := s.moods.ListBetween(uid, start, rangeEnd)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[moods] generate failed: %w", e)
	}
	journals, e := s.journals.ListBetween(uid, start, rangeEnd)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[journals] generate failed: %w", e)
	}

	data := buildWeeklySnapshot(start, end, moods, journals, time.Now())
	raw, e := json.Marshal(data)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[snapshot] generate failed: %w", e)
	}
	rec := &model.WeeklyReport{UserID: uid, EndDate: end, StartDate: start, Snapshot: string(raw)}
	inserted, e := s.repo.CreateIgnoreConflict(rec)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[end_date] create failed: %w", e)
	}
	if !inserted {
		// 并发场景：另一请求已先行落库，回读已有快照，不能覆盖。
		existing, e := s.repo.ByEndDate(uid, end)
		if e != nil {
			return nil, fmt.Errorf("WeeklyReport[end_date] refetch failed: %w", e)
		}
		s.logger.Info(constants.LogWeeklyReportReused, "user_id", uid, "end_date", end.Format(dayLayout))
		return s.toResponse(existing, true)
	}
	s.logger.Info(constants.LogWeeklyReportGenerated, "user_id", uid, "end_date", end.Format(dayLayout), "valid_days", data.ValidDays)
	return s.toResponse(rec, false)
}

// Latest 打开该账号最新一期已固化的周报。
func (s *WeeklyReportService) Latest(uid uint) (*dto.WeeklyReportResponse, error) {
	v, e := s.repo.Latest(uid)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[user_id] latest failed: %w", e)
	}
	return s.toResponse(v, true)
}

// GetByEndDate 按结束日打开已有周报。
func (s *WeeklyReportService) GetByEndDate(uid uint, endDate string) (*dto.WeeklyReportResponse, error) {
	end, e := resolveEndDate(endDate)
	if e != nil {
		return nil, util.NewAppError(constants.CodeValidation, "WeeklyReport[end_date] read failed: invalid date", e)
	}
	v, e := s.repo.ByEndDate(uid, end)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[end_date] read failed: %w", e)
	}
	return s.toResponse(v, true)
}

// Get 按周报编号打开，且仅限本人快照。
func (s *WeeklyReportService) Get(uid, id uint) (*dto.WeeklyReportResponse, error) {
	v, e := s.repo.ByID(id, uid)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[id=%d] read failed: %w", id, e)
	}
	return s.toResponse(v, true)
}

func (s *WeeklyReportService) toResponse(v *model.WeeklyReport, reused bool) (*dto.WeeklyReportResponse, error) {
	var data dto.WeeklyReportData
	if e := json.Unmarshal([]byte(v.Snapshot), &data); e != nil {
		return nil, fmt.Errorf("WeeklyReport[id=%d] snapshot parse failed: %w", v.ID, e)
	}
	return &dto.WeeklyReportResponse{
		ID:        v.ID,
		StartDate: v.StartDate.Format(dayLayout),
		EndDate:   v.EndDate.Format(dayLayout),
		Reused:    reused,
		CreatedAt: v.CreatedAt.Format(time.RFC3339),
		Report:    data,
	}, nil
}

func resolveEndDate(endDate string) (time.Time, error) {
	if endDate == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}
	d, e := time.Parse(dayLayout, endDate)
	if e != nil {
		return time.Time{}, e
	}
	return d.UTC(), nil
}

// buildWeeklySnapshot 按自然日聚合：每个自然日按记录时间（并列时按编号）
// 取最后一条情绪与最后一篇日记，统计平均心情、最低日、标签频次与有效天数。
func buildWeeklySnapshot(start, end time.Time, moods []model.Mood, journals []model.Journal, generatedAt time.Time) dto.WeeklyReportData {
	lastMoodByDay := map[string]model.Mood{}
	lastJournalByDay := map[string]model.Journal{}
	for _, m := range moods {
		key := m.RecordDate.UTC().Format(dayLayout)
		if prev, ok := lastMoodByDay[key]; !ok || moodAfter(m, prev) {
			lastMoodByDay[key] = m
		}
	}
	for _, j := range journals {
		key := j.CreatedAt.UTC().Format(dayLayout)
		if prev, ok := lastJournalByDay[key]; !ok || journalAfter(j, prev) {
			lastJournalByDay[key] = j
		}
	}

	days := make([]string, 0, WeeklyReportDays)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		days = append(days, d.Format(dayLayout))
	}

	entries := make([]dto.WeeklyReportDay, 0, len(days))
	tagHits := map[string]int{}
	sum, valid := 0, 0
	var lowest *dto.WeeklyReportDay
	for _, day := range days {
		m, ok := lastMoodByDay[day]
		if !ok {
			continue
		}
		entry := dto.WeeklyReportDay{Date: day, MoodLevel: m.MoodLevel}
		if j, has := lastJournalByDay[day]; has {
			entry.HasJournal = true
			entry.JournalTitle = j.Title
			entry.JournalMood = j.MoodLevel
		}
		entries = append(entries, entry)
		sum += m.MoodLevel
		valid++
		var tags []string
		if e := json.Unmarshal([]byte(m.MoodTags), &tags); e == nil {
			for _, t := range tags {
				tagHits[t]++
			}
		}
		if lowest == nil || entry.MoodLevel < lowest.MoodLevel {
			cp := entry
			lowest = &cp
		}
	}

	tagCounts := make([]dto.WeeklyReportTagCount, 0, len(tagHits))
	for tag, count := range tagHits {
		tagCounts = append(tagCounts, dto.WeeklyReportTagCount{Tag: tag, Count: count})
	}
	sort.Slice(tagCounts, func(i, j int) bool {
		if tagCounts[i].Count != tagCounts[j].Count {
			return tagCounts[i].Count > tagCounts[j].Count
		}
		return tagCounts[i].Tag < tagCounts[j].Tag
	})

	avg := 0.0
	if valid > 0 {
		avg = math.Round(float64(sum)/float64(valid)*10) / 10
	}
	return dto.WeeklyReportData{
		StartDate:   start.Format(dayLayout),
		EndDate:     end.Format(dayLayout),
		Average:     avg,
		LowestDay:   lowest,
		TagCounts:   tagCounts,
		ValidDays:   valid,
		Days:        entries,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
	}
}

// moodAfter 判断 a 是否晚于 b：先比记录时间，时间相同则编号更大者为“最后一条”。
func moodAfter(a, b model.Mood) bool {
	if a.RecordDate.Equal(b.RecordDate) {
		return a.ID > b.ID
	}
	return a.RecordDate.After(b.RecordDate)
}

// journalAfter 同理按创建时间与编号判断末篇日记。
func journalAfter(a, b model.Journal) bool {
	if a.CreatedAt.Equal(b.CreatedAt) {
		return a.ID > b.ID
	}
	return a.CreatedAt.After(b.CreatedAt)
}
