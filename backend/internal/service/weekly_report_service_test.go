package service

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"github.com/jackc/pgx/v5/pgconn"
)

// ---- fakes ----

type fakeMoodRangeRepo struct {
	moods []model.Mood
}

func (f *fakeMoodRangeRepo) Create(m *model.Mood) error {
	m.ID = uint(len(f.moods) + 1)
	f.moods = append(f.moods, *m)
	return nil
}
func (f *fakeMoodRangeRepo) List(uid uint, date *time.Time) ([]model.Mood, error) {
	return f.moods, nil
}
func (f *fakeMoodRangeRepo) ListRange(uid uint, start, end time.Time) (out []model.Mood, _ error) {
	for _, m := range f.moods {
		if !m.RecordDate.Before(start) && m.RecordDate.Before(end) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RecordDate.Equal(out[j].RecordDate) {
			return out[i].ID < out[j].ID
		}
		return out[i].RecordDate.Before(out[j].RecordDate)
	})
	return out, nil
}
func (f *fakeMoodRangeRepo) ByID(id, uid uint) (*model.Mood, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeMoodRangeRepo) Update(m *model.Mood) error { return nil }
func (f *fakeMoodRangeRepo) Delete(m *model.Mood) error { return nil }

type fakeJournalRangeRepo struct{ items []model.Journal }

func (f *fakeJournalRangeRepo) Create(j *model.Journal) error {
	j.ID = uint(len(f.items) + 1)
	f.items = append(f.items, *j)
	return nil
}
func (f *fakeJournalRangeRepo) List(uid uint, level int) ([]model.Journal, error) {
	return f.items, nil
}
func (f *fakeJournalRangeRepo) ListRange(uid uint, start, end time.Time) (out []model.Journal, _ error) {
	for _, j := range f.items {
		if !j.CreatedAt.Before(start) && j.CreatedAt.Before(end) {
			out = append(out, j)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}
func (f *fakeJournalRangeRepo) ByID(id, uid uint) (*model.Journal, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeJournalRangeRepo) Update(j *model.Journal) error { return nil }
func (f *fakeJournalRangeRepo) Delete(j *model.Journal) error { return nil }

type fakeWeeklyReportRepo struct {
	stored  map[string]*model.WeeklyReport
	nextID  uint
	failDup bool
}

func newFakeWeeklyReportRepo() *fakeWeeklyReportRepo {
	return &fakeWeeklyReportRepo{stored: map[string]*model.WeeklyReport{}}
}
func (f *fakeWeeklyReportRepo) key(uid uint, endDate string) string {
	return endDate
}
func (f *fakeWeeklyReportRepo) Create(v *model.WeeklyReport) error {
	key := f.key(v.UserID, v.EndDate.Format(constants.WeeklyReportDateLayout))
	if f.failDup {
		f.failDup = false
		return &pgconn.PgError{Code: "23505", Message: "duplicate key value"}
	}
	if _, ok := f.stored[key]; ok {
		return &pgconn.PgError{Code: "23505", Message: "duplicate key value"}
	}
	f.nextID++
	v.ID = f.nextID
	cp := *v
	f.stored[key] = &cp
	return nil
}
func (f *fakeWeeklyReportRepo) ByUserAndEndDate(uid uint, endDate string) (*model.WeeklyReport, error) {
	if v, ok := f.stored[f.key(uid, endDate)]; ok {
		return v, nil
	}
	return nil, repository.ErrNotFound
}

// ---- helpers ----

func testWeeklyService(r *fakeWeeklyReportRepo, m *fakeMoodRangeRepo, j *fakeJournalRangeRepo) *WeeklyReportService {
	return NewWeeklyReportService(r, m, j, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func mustTags(t *testing.T, tags ...string) string {
	t.Helper()
	b, e := json.Marshal(tags)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func dayAt(t *testing.T, date string, hour int) time.Time {
	t.Helper()
	d, e := time.Parse(constants.WeeklyReportDateLayout, date)
	if e != nil {
		t.Fatal(e)
	}
	return d.Add(time.Duration(hour) * time.Hour)
}

// ---- tests ----

func TestWeeklyReportGenerateAggregation(t *testing.T) {
	end := "2026-09-20"
	moods := &fakeMoodRangeRepo{}
	// 9/7（窗口外，早于起始日 9/7？起始日 = 9/7）当天边界：保留
	moods.moods = []model.Mood{
		{ID: 1, UserID: 1, MoodLevel: 4, MoodTags: mustTags(t, "anxious"), RecordDate: dayAt(t, "2026-09-07", 9)},
		{ID: 2, UserID: 1, MoodLevel: 6, MoodTags: mustTags(t, "calm", "happy"), RecordDate: dayAt(t, "2026-09-07", 20)}, // 当日末条
		{ID: 3, UserID: 1, MoodLevel: 9, MoodTags: mustTags(t, "happy"), RecordDate: dayAt(t, "2026-09-08", 8)},
		{ID: 4, UserID: 1, MoodLevel: 3, MoodTags: mustTags(t, "tired", "angry"), RecordDate: dayAt(t, "2026-09-10", 22)},
		{ID: 5, UserID: 1, MoodLevel: 5, MoodTags: mustTags(t, "tired"), RecordDate: dayAt(t, "2026-09-01", 10)}, // 窗口外
	}
	journals := &fakeJournalRangeRepo{}
	journals.items = []model.Journal{
		{ID: 1, UserID: 1, Title: "晨间随笔", MoodLevel: 5, CreatedAt: dayAt(t, "2026-09-07", 8)},
		{ID: 2, UserID: 1, Title: "晚风日记", MoodLevel: 7, CreatedAt: dayAt(t, "2026-09-07", 21)}, // 当日末篇
		{ID: 3, UserID: 1, Title: "没有情绪记录的一天", MoodLevel: 8, CreatedAt: dayAt(t, "2026-09-09", 20)},
	}
	s := testWeeklyService(newFakeWeeklyReportRepo(), moods, journals)

	rep, e := s.Generate(1, end)
	if e != nil {
		t.Fatalf("generate: %v", e)
	}
	if rep.AlreadyExist {
		t.Fatal("first generate must not be marked already_exists")
	}
	if rep.StartDate != "2026-09-07" || rep.EndDate != end {
		t.Fatalf("window = %s..%s", rep.StartDate, rep.EndDate)
	}
	if rep.ValidDays != 3 {
		t.Fatalf("valid days = %d, want 3", rep.ValidDays)
	}
	// 末条情绪：6、9、3 → 平均 6.0
	if rep.AverageMood != 6.0 {
		t.Fatalf("average = %v, want 6.0", rep.AverageMood)
	}
	if rep.LowestDate != "2026-09-10" || rep.LowestLevel != 3 {
		t.Fatalf("lowest = %s %d", rep.LowestDate, rep.LowestLevel)
	}
	// 标签频次：happy 2（9/7 末条 + 9/8），calm 1，anxious 0（非末条不计），tired 1，angry 1
	tagMap := map[string]int{}
	for _, tc := range rep.TagCounts {
		tagMap[tc.Tag] = tc.Count
	}
	if tagMap["happy"] != 2 || tagMap["calm"] != 1 || tagMap["tired"] != 1 || tagMap["angry"] != 1 || tagMap["anxious"] != 0 {
		t.Fatalf("tag counts = %v", tagMap)
	}
	// 9/7 条目必须取末条情绪 6，并带出同日末篇日记标题与心情
	var d7 *dto.WeeklyReportDay
	for i := range rep.Days {
		if rep.Days[i].Date == "2026-09-07" {
			d7 = &rep.Days[i]
		}
	}
	if d7 == nil || d7.MoodID != 2 || d7.MoodLevel != 6 {
		t.Fatalf("day 09-07 = %+v", d7)
	}
	if !d7.HasJournal || d7.JournalTitle != "晚风日记" || d7.JournalMood != 7 || d7.JournalID != 2 {
		t.Fatalf("journal of 09-07 = %+v", d7)
	}
	// 9/9 只有日记没有情绪，不进 days
	for _, d := range rep.Days {
		if d.Date == "2026-09-09" {
			t.Fatal("day without mood must not appear")
		}
	}
}

func TestWeeklyReportDuplicateGenerateDoesNotOverwrite(t *testing.T) {
	end := "2026-09-20"
	moods := &fakeMoodRangeRepo{moods: []model.Mood{
		{ID: 1, UserID: 1, MoodLevel: 8, MoodTags: mustTags(t, "happy"), RecordDate: dayAt(t, "2026-09-15", 10)},
	}}
	journals := &fakeJournalRangeRepo{}
	repo := newFakeWeeklyReportRepo()
	s := testWeeklyService(repo, moods, journals)

	first, e := s.Generate(1, end)
	if e != nil {
		t.Fatal(e)
	}
	// 事后改情绪/日记：快照固化后不回写
	moods.moods[0].MoodLevel = 2
	moods.moods[0].MoodTags = mustTags(t, "angry")

	second, e := s.Generate(1, end)
	if e != nil {
		t.Fatal(e)
	}
	if !second.AlreadyExist || second.ID != first.ID {
		t.Fatalf("duplicate generate must reuse snapshot: first=%d second=%d exist=%v", first.ID, second.ID, second.AlreadyExist)
	}
	if second.AverageMood != 8 {
		t.Fatalf("snapshot was overwritten: average=%v", second.AverageMood)
	}
	if len(second.TagCounts) != 1 || second.TagCounts[0].Tag != "happy" {
		t.Fatalf("snapshot tags changed: %+v", second.TagCounts)
	}
	if len(repo.stored) != 1 {
		t.Fatalf("only one snapshot allowed, got %d", len(repo.stored))
	}

	// GET 同样返回固化内容
	got, e := s.Get(1, end)
	if e != nil || got.AverageMood != 8 || !got.AlreadyExist {
		t.Fatalf("get = %+v, %v", got, e)
	}
}

func TestWeeklyReportConcurrentConflictReusesExisting(t *testing.T) {
	end := "2026-09-20"
	moods := &fakeMoodRangeRepo{moods: []model.Mood{
		{ID: 1, UserID: 1, MoodLevel: 7, MoodTags: mustTags(t, "calm"), RecordDate: dayAt(t, "2026-09-18", 10)},
	}}
	repo := newFakeWeeklyReportRepo()
	// 预置另一个并发请求已落库的快照
	repo.stored[end] = &model.WeeklyReport{ID: 99, UserID: 1, EndDate: dayAt(t, end, 0), AverageMood: 4.2, ValidDays: 2, Days: "[]", TagCounts: "[]"}
	repo.failDup = true
	s := testWeeklyService(repo, moods, &fakeJournalRangeRepo{})

	rep, e := s.Generate(1, end)
	if e != nil {
		t.Fatal(e)
	}
	if rep.ID != 99 || !rep.AlreadyExist || rep.AverageMood != 4.2 {
		t.Fatalf("conflict must return existing snapshot, got %+v", rep)
	}
	if len(repo.stored) != 1 {
		t.Fatalf("only one snapshot allowed, got %d", len(repo.stored))
	}
}

func TestWeeklyReportGetNotFoundAndValidation(t *testing.T) {
	s := testWeeklyService(newFakeWeeklyReportRepo(), &fakeMoodRangeRepo{}, &fakeJournalRangeRepo{})

	if _, e := s.Get(1, "2026-09-20"); !errors.Is(e, repository.ErrNotFound) {
		var app *util.AppError
		if !errors.As(e, &app) || app.Code != constants.CodeNotFound {
			t.Fatalf("want not found app error, got %v", e)
		}
	}
	if _, e := s.Generate(1, "20-09-2026"); e == nil {
		t.Fatal("invalid end_date must fail")
	}
}

func TestWeeklyReportEmptyWindow(t *testing.T) {
	s := testWeeklyService(newFakeWeeklyReportRepo(), &fakeMoodRangeRepo{}, &fakeJournalRangeRepo{})
	rep, e := s.Generate(1, "2026-09-20")
	if e != nil {
		t.Fatal(e)
	}
	if rep.ValidDays != 0 || rep.AverageMood != 0 || rep.LowestDate != "" || len(rep.Days) != 0 || len(rep.TagCounts) != 0 {
		t.Fatalf("empty window report = %+v", rep)
	}
}
