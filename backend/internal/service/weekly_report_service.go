package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"github.com/jackc/pgx/v5/pgconn"
)

type WeeklyReportService struct {
	repo     repository.WeeklyReportRepository
	moods    repository.MoodRepository
	journals repository.JournalRepository
	logger   *slog.Logger
}

func NewWeeklyReportService(r repository.WeeklyReportRepository, m repository.MoodRepository, j repository.JournalRepository, l *slog.Logger) *WeeklyReportService {
	return &WeeklyReportService{repo: r, moods: m, journals: j, logger: l}
}

// Generate 生成（或复用）结束日为 endDate 的 14 天周报。
// endDate 为空时取当天。重复生成直接返回已有快照，绝不覆盖。
func (s *WeeklyReportService) Generate(uid uint, endDate string) (*dto.WeeklyReportData, error) {
	end, e := s.resolveEndDate(endDate)
	if e != nil {
		return nil, e
	}
	endKey := end.Format(constants.WeeklyReportDateLayout)
	if existing, e := s.repo.ByUserAndEndDate(uid, endKey); e == nil {
		s.logger.Info(constants.LogWeeklyReportReused, "user_id", uid, "end_date", endKey)
		return s.toData(existing, true)
	} else if !errors.Is(e, repository.ErrNotFound) {
		return nil, fmt.Errorf("WeeklyReport[user_id] read failed: %w", e)
	}

	snapshot, e := s.buildSnapshot(uid, end)
	if e != nil {
		return nil, e
	}
	if e = s.repo.Create(snapshot); e != nil {
		if isDuplicateKey(e) {
			// 并发请求已先生成：回读已有快照，只保留一份。
			existing, rErr := s.repo.ByUserAndEndDate(uid, endKey)
			if rErr != nil {
				return nil, fmt.Errorf("WeeklyReport[end_date] read after conflict failed: %w", rErr)
			}
			s.logger.Info(constants.LogWeeklyReportReused, "user_id", uid, "end_date", endKey)
			return s.toData(existing, true)
		}
		return nil, fmt.Errorf("WeeklyReport[user_id] create failed: %w", e)
	}
	s.logger.Info(constants.LogWeeklyReportGenerated, "user_id", uid, "end_date", endKey, "valid_days", snapshot.ValidDays)
	return s.toData(snapshot, false)
}

// Get 读取已固化的当前周报快照；没有生成过时返回 repository.ErrNotFound。
func (s *WeeklyReportService) Get(uid uint, endDate string) (*dto.WeeklyReportData, error) {
	end, e := s.resolveEndDate(endDate)
	if e != nil {
		return nil, e
	}
	v, e := s.repo.ByUserAndEndDate(uid, end.Format(constants.WeeklyReportDateLayout))
	if e != nil {
		if errors.Is(e, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, "WeeklyReport[end_date] not found: snapshot has not been generated yet", e)
		}
		return nil, fmt.Errorf("WeeklyReport[user_id] read failed: %w", e)
	}
	s.logger.Info(constants.LogWeeklyReportRead, "user_id", uid, "end_date", end.Format(constants.WeeklyReportDateLayout))
	return s.toData(v, true)
}

func (s *WeeklyReportService) resolveEndDate(endDate string) (time.Time, error) {
	if strings.TrimSpace(endDate) == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}
	d, e := time.Parse(constants.WeeklyReportDateLayout, strings.TrimSpace(endDate))
	if e != nil {
		return time.Time{}, util.NewAppError(constants.CodeValidation, "WeeklyReport[end_date] generate failed: invalid date", e)
	}
	return d, nil
}

// buildSnapshot 取窗口内每个自然日按 (记录时间, id) 的最后一条情绪，
// 并带出同日按 (created_at, id) 的最后一篇日记标题与心情。
func (s *WeeklyReportService) buildSnapshot(uid uint, end time.Time) (*model.WeeklyReport, error) {
	start := end.AddDate(0, 0, 1-constants.WeeklyReportWindowDays)
	next := end.AddDate(0, 0, 1)

	moodList, e := s.moods.ListRange(uid, start, next)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[moods] aggregate failed: %w", e)
	}
	journalList, e := s.journals.ListRange(uid, start, next)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[journals] aggregate failed: %w", e)
	}

	lastJournalByDay := map[string]model.Journal{}
	for _, j := range journalList {
		lastJournalByDay[j.CreatedAt.UTC().Format(constants.WeeklyReportDateLayout)] = j
	}

	days := make([]dto.WeeklyReportDay, 0, constants.WeeklyReportWindowDays)
	tagCount := map[string]int{}
	sum, count := 0, 0
	lowestLevel := 0
	var lowestDate string

	for i := 0; i < constants.WeeklyReportWindowDays; i++ {
		day := start.AddDate(0, 0, i)
		dayKey := day.Format(constants.WeeklyReportDateLayout)
		var moodOfDay *model.Mood
		// ListRange 已按 (record_date, id) 升序，最后一条匹配自然日的记录即当日末条。
		for idx := range moodList {
			m := moodList[idx]
			if m.RecordDate.UTC().Format(constants.WeeklyReportDateLayout) == dayKey {
				moodOfDay = &moodList[idx]
			}
		}
		if moodOfDay == nil {
			continue
		}
		var tags []string
		if te := json.Unmarshal([]byte(moodOfDay.MoodTags), &tags); te != nil {
			tags = []string{}
		}
		for _, t := range tags {
			tagCount[t]++
		}
		dayRow := dto.WeeklyReportDay{
			Date:      dayKey,
			MoodID:    moodOfDay.ID,
			MoodLevel: moodOfDay.MoodLevel,
			MoodTags:  tags,
		}
		if j, ok := lastJournalByDay[dayKey]; ok {
			dayRow.JournalID = j.ID
			dayRow.JournalTitle = j.Title
			dayRow.JournalMood = j.MoodLevel
			dayRow.HasJournal = true
		}
		days = append(days, dayRow)

		sum += moodOfDay.MoodLevel
		count++
		// 最低心情日取最低值；并列时保留最早的自然日。
		if lowestLevel == 0 || moodOfDay.MoodLevel < lowestLevel {
			lowestLevel = moodOfDay.MoodLevel
			lowestDate = dayKey
		}
	}

	average := 0.0
	if count > 0 {
		average = math.Round(float64(sum)/float64(count)*10) / 10
	}

	tags := make([]dto.WeeklyReportTag, 0, len(tagCount))
	for t, c := range tagCount {
		tags = append(tags, dto.WeeklyReportTag{Tag: t, Count: c})
	}
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Tag < tags[j].Tag
	})

	daysJSON, e := json.Marshal(days)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[days] encode failed: %w", e)
	}
	tagsJSON, e := json.Marshal(tags)
	if e != nil {
		return nil, fmt.Errorf("WeeklyReport[tag_counts] encode failed: %w", e)
	}

	snapshot := &model.WeeklyReport{
		UserID:      uid,
		StartDate:   start,
		EndDate:     end,
		AverageMood: average,
		LowestLevel: lowestLevel,
		ValidDays:   count,
		Days:        string(daysJSON),
		TagCounts:   string(tagsJSON),
	}
	if lowestDate != "" {
		if d, pe := time.Parse(constants.WeeklyReportDateLayout, lowestDate); pe == nil {
			snapshot.LowestDate = &d
		}
	}
	return snapshot, nil
}

func (s *WeeklyReportService) toData(v *model.WeeklyReport, alreadyExist bool) (*dto.WeeklyReportData, error) {
	days := []dto.WeeklyReportDay{}
	tags := []dto.WeeklyReportTag{}
	if strings.TrimSpace(v.Days) != "" {
		if e := json.Unmarshal([]byte(v.Days), &days); e != nil {
			return nil, fmt.Errorf("WeeklyReport[days] decode failed: %w", e)
		}
	}
	if strings.TrimSpace(v.TagCounts) != "" {
		if e := json.Unmarshal([]byte(v.TagCounts), &tags); e != nil {
			return nil, fmt.Errorf("WeeklyReport[tag_counts] decode failed: %w", e)
		}
	}
	out := &dto.WeeklyReportData{
		ID:           v.ID,
		UserID:       v.UserID,
		StartDate:    v.StartDate.Format(constants.WeeklyReportDateLayout),
		EndDate:      v.EndDate.Format(constants.WeeklyReportDateLayout),
		AverageMood:  v.AverageMood,
		LowestLevel:  v.LowestLevel,
		ValidDays:    v.ValidDays,
		TagCounts:    tags,
		Days:         days,
		AlreadyExist: alreadyExist,
		CreatedAt:    v.CreatedAt.Format(time.RFC3339),
	}
	if v.LowestDate != nil {
		out.LowestDate = v.LowestDate.Format(constants.WeeklyReportDateLayout)
	}
	return out, nil
}

// isDuplicateKey 判断是否为 PostgreSQL unique_violation (SQLSTATE 23505)。
func isDuplicateKey(e error) bool {
	var pgErr *pgconn.PgError
	return errors.As(e, &pgErr) && pgErr.Code == "23505"
}
