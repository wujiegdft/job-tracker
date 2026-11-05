package tracker

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
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
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
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
	if _, err := s.db.Exec("PRAGMA user_version = 3"); err != nil {
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

func TestStoreDemoPersistenceAndClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Date(2026, 12, 31, 23, 0, 0, 0, time.Local)
	// Even a real application identical to a demo must never be cleared.
	real := DemoApplications(now)[0]
	real.IsDemo = false
	if err := s.Save(&real); err != nil {
		t.Fatal(err)
	}
	other := real
	other.ID, other.Company = 0, "手工录入的真实公司"
	if err := s.Save(&other); err != nil {
		t.Fatal(err)
	}
	// Neither direction of provenance can be changed by editing an existing row.
	real.IsDemo, real.Notes = true, "编辑真实申请，不能将其变成示例"
	if err := s.Save(&real); err != nil || real.IsDemo {
		t.Fatalf("real provenance after update=%v, err=%v", real.IsDemo, err)
	}
	wantReal, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if n, err := s.LoadDemo(now); err != nil || n != 18 {
		t.Fatalf("LoadDemo count=%d, err=%v", n, err)
	}
	items, err := s.List()
	if err != nil || len(items) != 20 {
		t.Fatalf("after loading: count=%d, err=%v", len(items), err)
	}
	oldIDs := make(map[int64]bool)
	var edited Application
	for _, a := range items {
		if a.IsDemo {
			oldIDs[a.ID] = true
			edited = a
		}
	}
	if len(oldIDs) != 18 {
		t.Fatalf("demo rows=%d, want 18", len(oldIDs))
	}
	edited.IsDemo, edited.Company, edited.Notes = false, "编辑后的示例公司", "保留示例身份"
	if err := s.Save(&edited); err != nil || !edited.IsDemo {
		t.Fatalf("demo provenance after update=%v, err=%v", edited.IsDemo, err)
	}
	if err := s.SetStatus(edited.ID, "Offer"); err != nil {
		t.Fatal(err)
	}
	edited.Status = "Offer"
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	items, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	foundEdited := false
	for _, a := range items {
		if a.ID == edited.ID {
			foundEdited = true
			if !reflect.DeepEqual(a, edited) {
				t.Fatalf("edited demo did not persist: %+v, want %+v", a, edited)
			}
		}
	}
	if !foundEdited {
		t.Fatal("edited demo missing after reopening")
	}
	// Replace old, including renamed/edited, fixtures and refresh their dates.
	for _, loadTime := range []time.Time{now.AddDate(0, 1, 0), now.AddDate(0, 1, 0)} {
		if n, err := s.LoadDemo(loadTime); err != nil || n != 18 {
			t.Fatalf("repeat LoadDemo count=%d, err=%v", n, err)
		}
		items, err = s.List()
		if err != nil || len(items) != 20 {
			t.Fatalf("repeat loading: count=%d, err=%v", len(items), err)
		}
		wantDemo := make(map[string]Application)
		for _, a := range DemoApplications(loadTime) {
			wantDemo[a.Company] = a
		}
		var gotReal []Application
		for _, a := range items {
			if !a.IsDemo {
				gotReal = append(gotReal, a)
				continue
			}
			if oldIDs[a.ID] {
				t.Fatalf("old demo ID %d was not replaced", a.ID)
			}
			a.ID = 0
			if want, ok := wantDemo[a.Company]; !ok || !reflect.DeepEqual(a, want) {
				t.Fatalf("replacement demo=%+v, want %+v", a, want)
			}
			delete(wantDemo, a.Company)
		}
		if len(wantDemo) != 0 || !reflect.DeepEqual(gotReal, wantReal) {
			t.Fatalf("missing demos=%v, real=%+v, want %+v", wantDemo, gotReal, wantReal)
		}
	}
	// The renamed demo is also removable without ever being reloaded.
	edited.ID = 0
	if err := s.Save(&edited); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.ClearDemo(); err != nil {
			t.Fatal(err)
		}
		items, err = s.List()
		if err != nil || !reflect.DeepEqual(items, wantReal) {
			t.Fatalf("ClearDemo real records=%+v, err=%v, want %+v", items, err, wantReal)
		}
	}
}

func TestStoreLoadDemoRollback(t *testing.T) {
	for _, phase := range []string{"delete", "insert", "commit"} {
		t.Run(phase, func(t *testing.T) {
			s, err := Open(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			now := time.Date(2024, 2, 29, 12, 0, 0, 0, time.Local)
			real := DemoApplications(now)[0]
			real.IsDemo = false
			if err := s.Save(&real); err != nil {
				t.Fatal(err)
			}
			if _, err := s.LoadDemo(now); err != nil {
				t.Fatal(err)
			}
			before, err := s.List()
			if err != nil {
				t.Fatal(err)
			}
			var fault string
			switch phase {
			case "delete":
				fault = `CREATE TRIGGER fail_demo BEFORE DELETE ON applications
					WHEN OLD.is_demo = 1 BEGIN SELECT RAISE(ABORT, 'test delete failure'); END`
			case "insert":
				// The seventh insert fails after deletion and six successful inserts.
				fault = `CREATE TRIGGER fail_demo BEFORE INSERT ON applications
					WHEN NEW.url = 'https://example.com/jobs/demo-07'
					BEGIN SELECT RAISE(ABORT, 'test partial insert failure'); END`
			case "commit":
				// All statements succeed, but a deferred foreign key rejects COMMIT.
				fault = `PRAGMA foreign_keys = ON;
					CREATE TABLE demo_parents (id INTEGER PRIMARY KEY);
					CREATE TABLE demo_children (id INTEGER PRIMARY KEY, parent_id INTEGER
						REFERENCES demo_parents(id) DEFERRABLE INITIALLY DEFERRED);
					CREATE TRIGGER fail_demo AFTER INSERT ON applications WHEN NEW.is_demo = 1
					BEGIN INSERT INTO demo_children VALUES (NEW.id, -1); END`
			}
			if _, err := s.db.Exec(fault); err != nil {
				t.Fatal(err)
			}
			if n, err := s.LoadDemo(now.AddDate(0, 0, 1)); err == nil || n != 0 {
				t.Fatalf("failed LoadDemo count=%d, err=%v", n, err)
			}
			after, err := s.List()
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("rollback changed records: after=%+v, err=%v, before=%+v", after, err, before)
			}
			if phase == "delete" {
				if err := s.ClearDemo(); err == nil {
					t.Fatal("ClearDemo hid a delete failure")
				}
			}
			if phase == "commit" {
				var children int
				if err := s.db.QueryRow("SELECT COUNT(*) FROM demo_children").Scan(&children); err != nil || children != 0 {
					t.Fatalf("trigger changes were not rolled back: count=%d, err=%v", children, err)
				}
			}
			if _, err := s.db.Exec("DROP TRIGGER fail_demo"); err != nil {
				t.Fatal(err)
			}
			if n, err := s.LoadDemo(now); err != nil || n != 18 {
				t.Fatalf("store unusable after rollback: count=%d, err=%v", n, err)
			}
		})
	}
}

func TestStoreMigratesOriginalV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	// This is the original v1 schema, not a v2 table with its version reset.
	_, err = db.Exec(`CREATE TABLE applications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		company TEXT NOT NULL, role TEXT NOT NULL, city TEXT NOT NULL,
		source TEXT NOT NULL, source_person TEXT NOT NULL, status TEXT NOT NULL,
		applied_date TEXT NOT NULL, follow_up_date TEXT NOT NULL,
		interview_at TEXT NOT NULL, url TEXT NOT NULL, jd TEXT NOT NULL,
		contact TEXT NOT NULL, notes TEXT NOT NULL, priority TEXT NOT NULL,
		salary TEXT NOT NULL
	);
	INSERT INTO applications VALUES (41, '星屿云研', '旧版 Go 工程师', '上海', '内推',
		'旧版来源人', '二面', '2024-02-29', '2024-03-01', '2024-03-02 14:00',
		'https://example.com/legacy', '旧版职位描述', '旧版联系人', '真实旧记录', '高', '30K');
	PRAGMA user_version = 1`)
	closeErr := db.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("create original v1: %v; close: %v", err, closeErr)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
		t.Fatalf("migrated version=%d, err=%v", version, err)
	}
	want := []Application{{ID: 41, Company: "星屿云研", Role: "旧版 Go 工程师", City: "上海",
		Source: "内推", SourcePerson: "旧版来源人", Status: "二面", AppliedDate: "2024-02-29",
		FollowUpDate: "2024-03-01", InterviewAt: "2024-03-02 14:00", URL: "https://example.com/legacy",
		JD: "旧版职位描述", Contact: "旧版联系人", Notes: "真实旧记录", Priority: "高", Salary: "30K"}}
	if items, err := s.List(); err != nil || !reflect.DeepEqual(items, want) {
		t.Fatalf("legacy fields/provenance changed: %+v, err=%v, want %+v", items, err, want)
	}
	if n, err := s.LoadDemo(time.Date(2024, 3, 1, 12, 0, 0, 0, time.Local)); err != nil || n != 18 {
		t.Fatalf("migrated LoadDemo count=%d, err=%v", n, err)
	}
	if err := s.ClearDemo(); err != nil {
		t.Fatal(err)
	}
	if items, err := s.List(); err != nil || !reflect.DeepEqual(items, want) {
		t.Fatalf("legacy record deleted: %+v, err=%v", items, err)
	}
	for _, bad := range []any{-1, 2, nil} {
		if _, err := s.db.Exec("UPDATE applications SET is_demo=? WHERE id=41", bad); err == nil {
			t.Fatalf("migrated schema accepted is_demo=%v", bad)
		}
	}
	// Opening an already migrated database is also idempotent.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if items, err := s.List(); err != nil || !reflect.DeepEqual(items, want) {
		t.Fatalf("reopened migrated store=%+v, err=%v", items, err)
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
