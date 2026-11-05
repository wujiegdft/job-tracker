package tracker

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDemoApplicationsCoverage(t *testing.T) {
	now := time.Date(2026, 11, 5, 23, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	items := DemoApplications(now)
	if len(items) != 18 {
		t.Fatalf("demo count=%d, want 18", len(items))
	}
	statuses, sources, priorities := map[string]int{}, map[string]int{}, map[string]int{}
	companies, cities := map[string]bool{}, map[string]bool{}
	for i, a := range items {
		if a.ID != 0 || !a.IsDemo {
			t.Fatalf("demo %d must be unsaved and marked: %+v", i, a)
		}
		if err := a.Validate(); err != nil {
			t.Fatalf("demo %d is invalid: %v", i, err)
		}
		if companies[a.Company] {
			t.Errorf("repeated company %q", a.Company)
		}
		companies[a.Company], cities[a.City] = true, true
		statuses[a.Status]++
		sources[a.Source]++
		priorities[a.Priority]++
		if (a.Source == "内推" || a.Source == "猎头") && a.SourcePerson == "" {
			t.Errorf("missing source person: %+v", a)
		}
		if a.Salary == "" || a.JD == "" || a.Contact == "" || !strings.Contains(a.Notes, "虚构") ||
			!strings.HasPrefix(a.URL, "https://example.com/") {
			t.Errorf("missing descriptive or fictional demo information: %+v", a)
		}
	}
	if len(statuses) != 9 || len(sources) != 6 || len(priorities) != 3 || len(cities) < 6 {
		t.Fatalf("coverage statuses=%v, sources=%v, priorities=%v, cities=%v", statuses, sources, priorities, cities)
	}
	for _, status := range Statuses {
		if statuses[status] != 2 {
			t.Errorf("status %q has %d cards, want 2", status, statuses[status])
		}
	}
	for _, source := range Sources {
		if sources[source] != 3 {
			t.Errorf("source %q has %d cards, want 3", source, sources[source])
		}
	}
	want := DemoApplications(now)
	items[0].Company, items[0].IsDemo = "changed", false
	if !reflect.DeepEqual(DemoApplications(now), want) {
		t.Fatal("demo results are not deterministic independent values")
	}
}

func TestDemoApplicationsRelativeCalendarDates(t *testing.T) {
	for _, tc := range []struct {
		name, localTime, today, yesterday, tomorrow string
	}{
		{"normal", "2026-11-05T23:59:00+08:00", "2026-11-05", "2026-11-04", "2026-11-06"},
		{"leap day next", "2024-02-28T00:01:00+14:00", "2024-02-28", "2024-02-27", "2024-02-29"},
		{"leap day today", "2024-02-29T23:59:00-12:00", "2024-02-29", "2024-02-28", "2024-03-01"},
		{"leap day previous", "2024-03-01T00:01:00+14:00", "2024-03-01", "2024-02-29", "2024-03-02"},
		{"non leap month", "2025-02-28T23:59:00-12:00", "2025-02-28", "2025-02-27", "2025-03-01"},
		{"next year", "2026-12-31T23:59:00-12:00", "2026-12-31", "2026-12-30", "2027-01-01"},
		{"previous year", "2027-01-01T00:01:00+14:00", "2027-01-01", "2026-12-31", "2027-01-02"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.localTime)
			if err != nil {
				t.Fatal(err)
			}
			items := DemoApplications(now)
			if items[0].AppliedDate != tc.today || items[1].AppliedDate != tc.yesterday {
				t.Fatalf("local applied dates=%q, %q; want %q, %q", items[0].AppliedDate, items[1].AppliedDate, tc.today, tc.yesterday)
			}
			var todayFollowUp, overdueFollowUp, tomorrowInterview bool
			for _, a := range items {
				if err := a.Validate(); err != nil {
					t.Fatalf("invalid calendar date: %+v: %v", a, err)
				}
				if a.AppliedDate > tc.today {
					t.Errorf("application is in the future: %+v", a)
				}
				todayFollowUp = todayFollowUp || a.FollowUpDate == tc.today
				overdueFollowUp = overdueFollowUp || a.FollowUpDate == tc.yesterday
				tomorrowInterview = tomorrowInterview || strings.HasPrefix(a.InterviewAt, tc.tomorrow+" ")
			}
			if !todayFollowUp || !overdueFollowUp || !tomorrowInterview {
				t.Fatalf("missing relative dates: today follow-up=%v, overdue=%v, tomorrow interview=%v", todayFollowUp, overdueFollowUp, tomorrowInterview)
			}
			midday := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
			if !reflect.DeepEqual(items, DemoApplications(midday)) {
				t.Fatal("time of day changed demo calendar dates")
			}
		})
	}
}
