package tracker

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

const applicationColumns = `company, role, city, source, source_person, status,
	applied_date, follow_up_date, interview_at, url, jd, contact, notes, priority, salary`

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("数据库路径不能为空")
	}
	dsn := path
	if path != ":memory:" {
		if err := ensureParentDirectory(path); err != nil {
			return nil, fmt.Errorf("创建数据目录失败: %w", err)
		}
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return nil, fmt.Errorf("打开数据库文件失败: %w", err)
		}
		chmodErr := f.Chmod(0600)
		closeErr := f.Close()
		if (chmodErr != nil && runtime.GOOS != "windows") || closeErr != nil {
			return nil, errors.New("设置数据库文件权限失败")
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		// A Windows drive belongs in the URI path, not its authority.
		absolute = filepath.ToSlash(absolute)
		if !strings.HasPrefix(absolute, "/") {
			absolute = "/" + absolute
		}
		// Escape filenames containing '?' or '#' instead of treating them as DSN options.
		dsn = (&url.URL{Scheme: "file", Path: absolute}).String()
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db: db}
	if _, err = db.Exec("PRAGMA busy_timeout = 5000"); err == nil {
		err = s.migrate()
	}
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > 1 {
		return errors.New("数据库版本高于当前程序支持的版本")
	}
	if version == 0 {
		if _, err := tx.Exec(`CREATE TABLE applications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company TEXT NOT NULL, role TEXT NOT NULL, city TEXT NOT NULL,
			source TEXT NOT NULL, source_person TEXT NOT NULL, status TEXT NOT NULL,
			applied_date TEXT NOT NULL, follow_up_date TEXT NOT NULL,
			interview_at TEXT NOT NULL, url TEXT NOT NULL, jd TEXT NOT NULL,
			contact TEXT NOT NULL, notes TEXT NOT NULL, priority TEXT NOT NULL,
			salary TEXT NOT NULL
		)`); err != nil {
			return err
		}
		if _, err := tx.Exec("PRAGMA user_version = 1"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Close() error {
	return s.db.Close()
}

// List returns newest applied dates first; IDs break ties in descending order.
// Empty dates sort last. An empty store returns a non-nil empty slice.
func (s *Store) List() ([]Application, error) {
	rows, err := s.db.Query("SELECT id, " + applicationColumns + " FROM applications ORDER BY applied_date DESC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applications := make([]Application, 0)
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.Company, &a.Role, &a.City, &a.Source,
			&a.SourcePerson, &a.Status, &a.AppliedDate, &a.FollowUpDate,
			&a.InterviewAt, &a.URL, &a.JD, &a.Contact, &a.Notes, &a.Priority,
			&a.Salary); err != nil {
			return nil, err
		}
		applications = append(applications, a)
	}
	return applications, rows.Err()
}

// Save validates a manual/confirmed record. ID zero inserts; a positive ID
// updates an existing record. AI drafts are never saved implicitly.
func (s *Store) Save(a *Application) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.ID < 0 {
		return errors.New("申请 ID 无效")
	}
	args := []any{a.Company, a.Role, a.City, a.Source, a.SourcePerson, a.Status,
		a.AppliedDate, a.FollowUpDate, a.InterviewAt, a.URL, a.JD, a.Contact,
		a.Notes, a.Priority, a.Salary}
	if a.ID == 0 {
		result, err := s.db.Exec("INSERT INTO applications ("+applicationColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", args...)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		a.ID = id
		return nil
	}
	args = append(args, a.ID)
	result, err := s.db.Exec(`UPDATE applications SET company=?, role=?, city=?,
		source=?, source_person=?, status=?, applied_date=?, follow_up_date=?,
		interview_at=?, url=?, jd=?, contact=?, notes=?, priority=?, salary=? WHERE id=?`, args...)
	return requireExisting(result, err)
}

func (s *Store) Delete(id int64) error {
	result, err := s.db.Exec("DELETE FROM applications WHERE id = ?", id)
	return requireExisting(result, err)
}

func (s *Store) SetStatus(id int64, status string) error {
	status = strings.TrimSpace(status)
	if !contains(Statuses, status) {
		return errors.New("申请状态无效")
	}
	result, err := s.db.Exec("UPDATE applications SET status = ? WHERE id = ?", status, id)
	return requireExisting(result, err)
}

func requireExisting(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("申请记录不存在")
	}
	return nil
}
