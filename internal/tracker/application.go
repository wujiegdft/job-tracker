// Package tracker provides local application storage and optional AI-assisted
// drafts. It has no dependency on the desktop UI.
package tracker

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Application struct {
	ID           int64  `json:"id"`
	Company      string `json:"company"`
	Role         string `json:"role"`
	City         string `json:"city"`
	Source       string `json:"source"`
	SourcePerson string `json:"source_person"`
	Status       string `json:"status"`
	AppliedDate  string `json:"applied_date"`
	FollowUpDate string `json:"follow_up_date"`
	InterviewAt  string `json:"interview_at"`
	URL          string `json:"url"`
	JD           string `json:"jd"`
	Contact      string `json:"contact"`
	Notes        string `json:"notes"`
	Priority     string `json:"priority"`
	Salary       string `json:"salary"`
}

var Statuses = []string{"已投递", "简历筛选", "笔试", "一面", "二面", "HR面", "Offer", "被拒", "放弃"}
var Sources = []string{"内推", "Boss直聘", "猎聘", "猎头", "官网", "其他"}
var Priorities = []string{"普通", "高", "低"}

func NewApplication() Application {
	return Application{
		AppliedDate: time.Now().In(time.Local).Format(time.DateOnly),
		Status:      "已投递",
		Priority:    "普通",
		Source:      "其他",
	}
}

func (a *Application) trim() {
	for _, field := range []*string{
		&a.Company, &a.Role, &a.City, &a.Source, &a.SourcePerson, &a.Status,
		&a.AppliedDate, &a.FollowUpDate, &a.InterviewAt, &a.URL, &a.JD,
		&a.Contact, &a.Notes, &a.Priority, &a.Salary,
	} {
		*field = strings.TrimSpace(*field)
	}
}

func (a *Application) Validate() error {
	if a == nil {
		return errors.New("申请记录不能为空")
	}
	a.trim()
	if a.Company == "" || a.Role == "" {
		return errors.New("公司和岗位为必填项")
	}
	for _, date := range []struct{ name, value, layout string }{
		{"投递日期", a.AppliedDate, time.DateOnly},
		{"跟进日期", a.FollowUpDate, time.DateOnly},
		{"面试时间", a.InterviewAt, "2006-01-02 15:04"},
	} {
		if date.value == "" {
			continue
		}
		parsed, err := time.ParseInLocation(date.layout, date.value, time.Local)
		if err != nil || parsed.Format(date.layout) != date.value {
			return fmt.Errorf("%s格式无效", date.name)
		}
	}
	if a.URL != "" && !validHTTPURL(a.URL) {
		return errors.New("职位链接必须为有效的 HTTP(S) 地址")
	}
	if !contains(Statuses, a.Status) {
		return errors.New("申请状态无效")
	}
	if !contains(Priorities, a.Priority) {
		return errors.New("优先级无效")
	}
	if (a.Source == "内推" || a.Source == "猎头") && a.SourcePerson == "" {
		return errors.New("内推或猎头来源需要填写来源人")
	}
	return nil
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func validHTTPURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}

type Filter struct {
	Company string
	Role    string
	City    string
	Source  string
	Status  string
}

func Matches(a Application, f Filter) bool {
	return strings.Contains(strings.ToLower(a.Company), strings.ToLower(f.Company)) &&
		strings.Contains(strings.ToLower(a.Role), strings.ToLower(f.Role)) &&
		strings.Contains(strings.ToLower(a.City), strings.ToLower(f.City)) &&
		(f.Source == "" || a.Source == f.Source) &&
		(f.Status == "" || a.Status == f.Status)
}
