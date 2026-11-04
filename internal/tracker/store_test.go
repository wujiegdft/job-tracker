package tracker

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestStoreWindowsDrivePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("requires a Windows drive path")
	}
	path := filepath.Join(t.TempDir(), "Users", "wujie", "AppData", "Roaming", "job-tracker", "applications.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if items, err := s.List(); err != nil || len(items) != 0 {
		t.Fatalf("new Windows store: %v, %v", items, err)
	}
}

func TestStoreCRUDPersistence(t *testing.T) {
	filename := "jobs?#.sqlite"
	if runtime.GOOS == "windows" {
		// '?' is not a valid Windows filename; '#' and spaces still need URI escaping.
		filename = "jobs #.sqlite"
	}
	path := filepath.Join(t.TempDir(), "private", filename)
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	checkMode(t, filepath.Dir(path), 0700)
	checkMode(t, path, 0600)
	var version, timeout int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 1 {
		t.Fatalf("migration version=%d, err=%v", version, err)
	}
	if err := s.db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil || timeout != 5000 || s.db.Stats().MaxOpenConnections != 1 {
		t.Fatalf("busy timeout=%d, err=%v, pool=%+v", timeout, err, s.db.Stats())
	}
	if list, err := s.List(); err != nil || list == nil || len(list) != 0 {
		t.Fatalf("empty List()=%v, %v", list, err)
	}
	a := Application{
		Company: "公司'; DROP TABLE applications; --", Role: "Go 工程师", City: "上海",
		Source: "猎头", SourcePerson: "李先生", Status: "简历筛选", AppliedDate: "2026-11-01",
		FollowUpDate: "2026-11-04", InterviewAt: "2026-11-05 10:00", URL: "https://example.com/职位",
		JD: "Go 开发\n原文第二行", Contact: "招聘联系人", Notes: "不要丢失备注", Priority: "高", Salary: "30-40K",
	}
	if err := s.Save(&a); err != nil || a.ID <= 0 {
		t.Fatalf("insert ID=%d, err=%v", a.ID, err)
	}
	a.Role, a.Notes, a.Priority = "资深 Go 工程师", "修改后的备注", "普通"
	if err := s.Save(&a); err != nil {
		t.Fatal(err)
	}
	if err := s.SetStatus(a.ID, " 一面 "); err != nil {
		t.Fatal(err)
	}
	a.Status = "一面"
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	list, err := s.List()
	if err != nil || !reflect.DeepEqual(list, []Application{a}) {
		t.Fatalf("persisted List()=%+v, err=%v, want %+v", list, err, a)
	}
	if err := s.SetStatus(a.ID, "' OR 1=1; --"); err == nil {
		t.Fatal("invalid status accepted")
	}
	invalid := a
	invalid.Company = " "
	if err := s.Save(&invalid); err == nil {
		t.Fatal("invalid record saved")
	}
	if err := s.Save(nil); err == nil {
		t.Fatal("nil record saved")
	}
	missing := a
	missing.ID += 100
	if err := s.Save(&missing); err == nil {
		t.Fatal("update silently inserted missing record")
	}
	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(a.ID); err == nil {
		t.Fatal("delete should report a missing record")
	}
	if err := s.SetStatus(a.ID, "Offer"); err == nil {
		t.Fatal("status update should report a missing record")
	}
	if list, err := s.List(); err != nil || len(list) != 0 {
		t.Fatalf("List after deletion=%v, %v", list, err)
	}
}

func TestStoreOrderAndMultidimensionalFilters(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Each dimension independently rules out a different near-match. This also
	// catches accidentally swapping company/role/city or using OR instead of AND.
	fixtures := []Application{
		{Company: "Alpha Labs", Role: "Go Backend", City: "ShangHai", Source: "Boss直聘", Status: "一面", AppliedDate: "2026-11-02"},
		{Company: "Beta Labs", Role: "Go Backend", City: "ShangHai", Source: "Boss直聘", Status: "一面", AppliedDate: "2026-11-03"},
		{Company: "Alpha Labs", Role: "Java Developer", City: "ShangHai", Source: "Boss直聘", Status: "一面", AppliedDate: "2026-11-02"},
		{Company: "Alpha Labs", Role: "Go Backend", City: "BeiJing", Source: "Boss直聘", Status: "一面", AppliedDate: "2026-11-01"},
		{Company: "Alpha Labs", Role: "Go Backend", City: "ShangHai", Source: "官网", Status: "一面", AppliedDate: ""},
		{Company: "Alpha Labs", Role: "Go Backend", City: "ShangHai", Source: "Boss直聘", Status: "Offer", AppliedDate: "2026-11-02"},
	}
	for i := range fixtures {
		fixtures[i].Priority = "普通"
		if err := s.Save(&fixtures[i]); err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	if !reflect.DeepEqual(ids, []int64{2, 6, 3, 1, 4, 5}) {
		t.Fatalf("date/ID order=%v", ids)
	}
	filter := Filter{Company: "aLpHa", Role: "GO back", City: "anghai", Source: "Boss直聘", Status: "一面"}
	var matched []int64
	for _, a := range list {
		if !Matches(a, Filter{}) {
			t.Fatal("empty filter should match every record")
		}
		if Matches(a, filter) {
			matched = append(matched, a.ID)
		}
	}
	if !reflect.DeepEqual(matched, []int64{1}) {
		t.Fatalf("combined filter matched IDs %v, want only 1", matched)
	}
	if Matches(fixtures[0], Filter{Source: "boss直聘"}) || Matches(fixtures[0], Filter{Status: "一"}) {
		t.Fatal("source/status matching must be exact")
	}
}

func TestStoreRejectsFutureSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("PRAGMA user_version = 2"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("future schema accepted")
	}
}

func checkMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return // Unix permission bits do not model Windows ACLs.
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode=%#o, want %#o", path, got, want)
	}
}
