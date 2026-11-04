package tracker

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestApplicationDefaultsAndJSON(t *testing.T) {
	before := time.Now().Format(time.DateOnly)
	a := NewApplication()
	after := time.Now().Format(time.DateOnly)
	if (a.AppliedDate != before && a.AppliedDate != after) || a.ID != 0 || a.Status != "已投递" || a.Priority != "普通" || a.Source != "其他" {
		t.Fatalf("unexpected defaults: %+v", a)
	}
	wantTags := []string{"id", "company", "role", "city", "source", "source_person", "status", "applied_date", "follow_up_date", "interview_at", "url", "jd", "contact", "notes", "priority", "salary"}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != len(wantTags) {
		t.Fatalf("JSON has %d fields, want %d", len(fields), len(wantTags))
	}
	for _, tag := range wantTags {
		if _, ok := fields[tag]; !ok {
			t.Errorf("missing JSON field %s", tag)
		}
	}
	if !reflect.DeepEqual(Statuses, []string{"已投递", "简历筛选", "笔试", "一面", "二面", "HR面", "Offer", "被拒", "放弃"}) ||
		!reflect.DeepEqual(Sources, []string{"内推", "Boss直聘", "猎聘", "猎头", "官网", "其他"}) ||
		!reflect.DeepEqual(Priorities, []string{"普通", "高", "低"}) {
		t.Fatal("public options differ from the UI contract")
	}
}

func TestApplicationValidate(t *testing.T) {
	valid := NewApplication()
	valid.Company, valid.Role = "  公司\t", " Go 工程师 "
	valid.AppliedDate, valid.FollowUpDate, valid.InterviewAt = "2024-02-29", "2026-11-05", "2026-11-06 09:30"
	valid.URL, valid.Source = " https://example.com/jobs/1 ", "校园招聘"
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	if valid.Company != "公司" || valid.Role != "Go 工程师" || valid.URL != "https://example.com/jobs/1" {
		t.Fatal("validation did not trim fields")
	}
	for _, tc := range []struct {
		name   string
		change func(*Application)
		valid  bool
	}{
		{"empty company", func(a *Application) { a.Company = " \n " }, false},
		{"empty role", func(a *Application) { a.Role = "" }, false},
		{"empty dates", func(a *Application) { a.AppliedDate, a.FollowUpDate, a.InterviewAt = "", "", "" }, true},
		{"invalid calendar date", func(a *Application) { a.AppliedDate = "2025-02-29" }, false},
		{"noncanonical date", func(a *Application) { a.FollowUpDate = "2026-1-05" }, false},
		{"noncanonical time", func(a *Application) { a.InterviewAt = "2026-11-06 9:30" }, false},
		{"time with seconds", func(a *Application) { a.InterviewAt = "2026-11-06 09:30:00" }, false},
		{"invalid time", func(a *Application) { a.InterviewAt = "2026-11-06 24:00" }, false},
		{"invalid URL scheme", func(a *Application) { a.URL = "javascript:alert(1)" }, false},
		{"relative URL", func(a *Application) { a.URL = "/job" }, false},
		{"missing URL host", func(a *Application) { a.URL = "https:///job" }, false},
		{"empty URL", func(a *Application) { a.URL = "" }, true},
		{"HTTP URL", func(a *Application) { a.URL = "http://localhost:8000/job" }, true},
		{"invalid status", func(a *Application) { a.Status = "成功" }, false},
		{"invalid priority", func(a *Application) { a.Priority = "紧急" }, false},
		{"referral without person", func(a *Application) { a.Source = "内推"; a.SourcePerson = " " }, false},
		{"headhunter without person", func(a *Application) { a.Source = "猎头" }, false},
		{"referral with person", func(a *Application) { a.Source = "内推"; a.SourcePerson = " 张三 " }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := valid
			tc.change(&a)
			if err := a.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate() = %v, want valid=%v", err, tc.valid)
			}
		})
	}
}
