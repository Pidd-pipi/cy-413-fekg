package service

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
)

type fakeWeeklyRepo struct {
	rows map[[2]uint]*model.WeeklyReport // [user,endUnix]
	next uint
}

func newFakeWeeklyRepo() *fakeWeeklyRepo {
	return &fakeWeeklyRepo{rows: map[[2]uint]*model.WeeklyReport{}}
}
func weeklyKey(uid uint, end time.Time) [2]uint {
	return [2]uint{uid, uint(end.UTC().Truncate(24 * time.Hour).Unix())}
}
func (f *fakeWeeklyRepo) CreateIgnoreConflict(v *model.WeeklyReport) (bool, error) {
	k := weeklyKey(v.UserID, v.EndDate)
	if _, ok := f.rows[k]; ok {
		return false, nil
	}
	f.next++
	v.ID = f.next
	f.rows[k] = v
	return true, nil
}
func (f *fakeWeeklyRepo) ByEndDate(uid uint, end time.Time) (*model.WeeklyReport, error) {
	if v, ok := f.rows[weeklyKey(uid, end)]; ok {
		return v, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeWeeklyRepo) ByID(id, uid uint) (*model.WeeklyReport, error) {
	for _, v := range f.rows {
		if v.ID == id && v.UserID == uid {
			return v, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (f *fakeWeeklyRepo) Latest(uid uint) (*model.WeeklyReport, error) {
	var best *model.WeeklyReport
	for _, v := range f.rows {
		if v.UserID != uid {
			continue
		}
		if best == nil || v.EndDate.After(best.EndDate) {
			best = v
		}
	}
	if best == nil {
		return nil, repository.ErrNotFound
	}
	return best, nil
}

type fakeMoodRangeRepo struct{ items []model.Mood }

func (f *fakeMoodRangeRepo) Create(*model.Mood) error                    { return nil }
func (f *fakeMoodRangeRepo) List(uint, *time.Time) ([]model.Mood, error) { return nil, nil }
func (f *fakeMoodRangeRepo) ListBetween(uid uint, start, end time.Time) (out []model.Mood, e error) {
	for _, m := range f.items {
		if m.UserID == uid && !m.RecordDate.Before(start) && m.RecordDate.Before(end) {
			out = append(out, m)
		}
	}
	return
}
func (f *fakeMoodRangeRepo) ByID(uint, uint) (*model.Mood, error) { return nil, repository.ErrNotFound }
func (f *fakeMoodRangeRepo) Update(*model.Mood) error             { return nil }
func (f *fakeMoodRangeRepo) Delete(*model.Mood) error             { return nil }

type fakeJournalRangeRepo struct{ items []model.Journal }

func (f *fakeJournalRangeRepo) Create(*model.Journal) error             { return nil }
func (f *fakeJournalRangeRepo) List(uint, int) ([]model.Journal, error) { return nil, nil }
func (f *fakeJournalRangeRepo) ListBetween(uid uint, start, end time.Time) (out []model.Journal, e error) {
	for _, j := range f.items {
		if j.UserID == uid && !j.CreatedAt.Before(start) && j.CreatedAt.Before(end) {
			out = append(out, j)
		}
	}
	return
}
func (f *fakeJournalRangeRepo) ByID(uint, uint) (*model.Journal, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeJournalRangeRepo) Update(*model.Journal) error { return nil }
func (f *fakeJournalRangeRepo) Delete(*model.Journal) error { return nil }

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func mustDate(t *testing.T, s string) time.Time {
	d, e := time.Parse(dayLayout, s)
	if e != nil {
		t.Fatalf("parse %s: %v", s, e)
	}
	return d.UTC()
}

func TestBuildWeeklySnapshot(t *testing.T) {
	end := mustDate(t, "2026-09-23")
	start := end.AddDate(0, 0, -(WeeklyReportDays - 1))
	day1 := mustDate(t, "2026-09-10")
	day2 := mustDate(t, "2026-09-23")
	cases := []struct {
		name      string
		moods     []model.Mood
		journals  []model.Journal
		valid     int
		avg       float64
		lowest    int
		topTag    string
		lastTitle string
	}{
		{
			name: "picks last mood and last journal of each calendar day",
			moods: []model.Mood{
				{ID: 1, UserID: 1, MoodLevel: 3, MoodTags: `["anxious"]`, RecordDate: day1.Add(8 * time.Hour)},
				{ID: 2, UserID: 1, MoodLevel: 8, MoodTags: `["happy","calm"]`, RecordDate: day1.Add(20 * time.Hour)},
				{ID: 3, UserID: 1, MoodLevel: 5, MoodTags: `["happy"]`, RecordDate: day2.Add(9 * time.Hour)},
			},
			journals: []model.Journal{
				{ID: 1, UserID: 1, Title: "早晨日记", MoodLevel: 6, CreatedAt: day1.Add(7 * time.Hour)},
				{ID: 2, UserID: 1, Title: "末篇日记", MoodLevel: 9, CreatedAt: day1.Add(22 * time.Hour)},
			},
			valid: 2, avg: 6.5, lowest: 5, topTag: "happy", lastTitle: "末篇日记",
		},
		{
			name:   "empty window",
			moods:  nil,
			valid:  0,
			avg:    0,
			lowest: -1,
			topTag: "",
		},
		{
			name: "same record time broken by id",
			moods: []model.Mood{
				{ID: 7, UserID: 1, MoodLevel: 4, MoodTags: `["tired"]`, RecordDate: day2.Add(10 * time.Hour)},
				{ID: 9, UserID: 1, MoodLevel: 7, MoodTags: `["calm"]`, RecordDate: day2.Add(10 * time.Hour)},
			},
			valid: 1, avg: 7, lowest: 7, topTag: "calm",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWeeklySnapshot(start, end, tt.moods, tt.journals, end)
			if got.ValidDays != tt.valid {
				t.Fatalf("valid_days = %d want %d", got.ValidDays, tt.valid)
			}
			if got.Average != tt.avg {
				t.Fatalf("average = %v want %v", got.Average, tt.avg)
			}
			if tt.valid == 0 {
				if got.LowestDay != nil {
					t.Fatalf("lowest_day must be nil for empty window, got %+v", got.LowestDay)
				}
				if len(got.TagCounts) != 0 || len(got.Days) != 0 {
					t.Fatalf("empty window must have empty counts/days")
				}
				return
			}
			if got.LowestDay == nil || got.LowestDay.MoodLevel != tt.lowest {
				t.Fatalf("lowest = %+v want level %d", got.LowestDay, tt.lowest)
			}
			if tt.topTag != "" && (len(got.TagCounts) == 0 || got.TagCounts[0].Tag != tt.topTag) {
				t.Fatalf("top tag = %+v want %s", got.TagCounts, tt.topTag)
			}
			if tt.lastTitle != "" {
				var title string
				for _, d := range got.Days {
					if d.HasJournal {
						title = d.JournalTitle
					}
				}
				if title != tt.lastTitle {
					t.Fatalf("last journal title = %q want %q", title, tt.lastTitle)
				}
			}
			if got.StartDate != start.Format(dayLayout) || got.EndDate != end.Format(dayLayout) {
				t.Fatalf("window = %s..%s", got.StartDate, got.EndDate)
			}
		})
	}
}

func TestWeeklyReportGenerateIsIdempotentAndFrozen(t *testing.T) {
	const uid = uint(42)
	end := "2026-09-23"
	endTime := mustDate(t, end)
	moods := &fakeMoodRangeRepo{items: []model.Mood{
		{ID: 1, UserID: uid, MoodLevel: 7, MoodTags: `["happy"]`, RecordDate: endTime.Add(12 * time.Hour)},
	}}
	journals := &fakeJournalRangeRepo{}
	repo := newFakeWeeklyRepo()
	svc := NewWeeklyReportService(repo, moods, journals, discardLogger())

	first, e := svc.Generate(uid, end)
	if e != nil || first.Reused {
		t.Fatalf("first generate: reused=%v err=%v", first, e)
	}
	if first.Report.Average != 7 || first.Report.ValidDays != 1 {
		t.Fatalf("unexpected first snapshot: %+v", first.Report)
	}

	// 基础数据变化后重复生成：必须返回已有快照（reused=true，不覆盖、不回写）。
	moods.items[0].MoodLevel = 2
	moods.items[0].MoodTags = `["angry"]`
	second, e := svc.Generate(uid, end)
	if e != nil {
		t.Fatalf("second generate: %v", e)
	}
	if !second.Reused || second.ID != first.ID {
		t.Fatalf("duplicate generate must reuse snapshot: %+v", second)
	}
	if second.Report.Average != 7 {
		t.Fatalf("frozen snapshot was rewritten: average=%v", second.Report.Average)
	}
	if len(repo.rows) != 1 {
		t.Fatalf("concurrent/duplicate generation must keep a single row, got %d", len(repo.rows))
	}

	// 并发同一结束日落库时只保留一份，冲突方回读。
	rec := &model.WeeklyReport{UserID: uid, EndDate: endTime, StartDate: endTime, Snapshot: `{}`}
	inserted, e := repo.CreateIgnoreConflict(rec)
	if e != nil || inserted {
		t.Fatalf("conflicting insert must be ignored, inserted=%v err=%v", inserted, e)
	}

	// latest / by id 能打开同一份快照。
	latest, e := svc.Latest(uid)
	if e != nil || latest.ID != first.ID {
		t.Fatalf("latest = %+v err=%v", latest, e)
	}
	byID, e := svc.Get(uid, first.ID)
	if e != nil || byID.EndDate != end {
		t.Fatalf("get by id = %+v err=%v", byID, e)
	}
}

func TestWeeklyReportRejectsInvalidEndDate(t *testing.T) {
	svc := NewWeeklyReportService(newFakeWeeklyRepo(), &fakeMoodRangeRepo{}, &fakeJournalRangeRepo{}, discardLogger())
	if _, e := svc.Generate(1, "2026-9-23"); e == nil {
		t.Fatal("invalid end_date must be rejected")
	}
}
