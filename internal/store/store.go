package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nissyi-gh/flow/internal/model"
	_ "modernc.org/sqlite"
)

// TaskStore manages SQLite persistence for tasks.
type TaskStore struct {
	db *sql.DB
}

func defaultDBPath() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(dataHome, "flow")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "flow.db"), nil
}

// NewTaskStore opens (or creates) the SQLite database and ensures the schema exists.
func NewTaskStore(dbPath string) (*TaskStore, error) {
	if dbPath == "" {
		var err error
		dbPath, err = defaultDBPath()
		if err != nil {
			return nil, fmt.Errorf("determine db path: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	schema := `CREATE TABLE IF NOT EXISTS tasks (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		title      TEXT    NOT NULL,
		completed  INTEGER NOT NULL DEFAULT 0,
		created_at TEXT    NOT NULL DEFAULT (datetime('now'))
	)`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	if err := migrateParentID(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate parent_id: %w", err)
	}

	if err := migrateScheduledOn(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate scheduled_on: %w", err)
	}

	if err := migrateDueDate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate due_date: %w", err)
	}

	return &TaskStore{db: db}, nil
}

func migrateParentID(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tasks)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasParentID := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "parent_id" {
			hasParentID = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasParentID {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN parent_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE")
		return err
	}
	return nil
}

func migrateScheduledOn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tasks)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasScheduledOn := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "scheduled_on" {
			hasScheduledOn = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasScheduledOn {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN scheduled_on TEXT")
		return err
	}
	return nil
}

func migrateDueDate(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(tasks)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasDueDate := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "due_date" {
			hasDueDate = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasDueDate {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN due_date TEXT")
		return err
	}
	return nil
}

func scanTask(scanner interface{ Scan(...any) error }) (model.Task, error) {
	var t model.Task
	var comp int
	var createdStr string
	var parentID sql.NullInt64
	var scheduledOn sql.NullString
	var dueDate sql.NullString
	if err := scanner.Scan(&t.ID, &t.Title, &comp, &createdStr, &parentID, &scheduledOn, &dueDate); err != nil {
		return model.Task{}, err
	}
	t.Completed = comp != 0
	t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
	if parentID.Valid {
		pid := int(parentID.Int64)
		t.ParentID = &pid
	}
	if scheduledOn.Valid {
		s := scheduledOn.String
		t.ScheduledOn = &s
	}
	if dueDate.Valid {
		d := dueDate.String
		t.DueDate = &d
	}
	return t, nil
}

// Add inserts a new task and returns it. parentID can be nil for root tasks.
func (s *TaskStore) Add(title string, parentID *int) (model.Task, error) {
	var res sql.Result
	var err error
	if parentID != nil {
		res, err = s.db.Exec("INSERT INTO tasks (title, parent_id) VALUES (?, ?)", title, *parentID)
	} else {
		res, err = s.db.Exec("INSERT INTO tasks (title) VALUES (?)", title)
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("insert task: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(int(id))
}

// List returns all tasks ordered by creation date ascending.
func (s *TaskStore) List() ([]model.Task, error) {
	rows, err := s.db.Query("SELECT id, title, completed, created_at, parent_id, scheduled_on, due_date FROM tasks ORDER BY created_at ASC")
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetByID retrieves a single task by its ID.
func (s *TaskStore) GetByID(id int) (model.Task, error) {
	row := s.db.QueryRow("SELECT id, title, completed, created_at, parent_id, scheduled_on, due_date FROM tasks WHERE id = ?", id)
	t, err := scanTask(row)
	if err != nil {
		return model.Task{}, fmt.Errorf("get task %d: %w", id, err)
	}
	return t, nil
}

// ToggleComplete atomically flips the completed status of a task.
func (s *TaskStore) ToggleComplete(id int) error {
	_, err := s.db.Exec("UPDATE tasks SET completed = 1 - completed WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("toggle task %d: %w", id, err)
	}
	return nil
}

// ToggleToday toggles the scheduled_on date for today.
// If scheduled_on is already today, it clears it; otherwise sets it to today.
func (s *TaskStore) ToggleToday(id int) error {
	today := time.Now().Format("2006-01-02")
	_, err := s.db.Exec(
		"UPDATE tasks SET scheduled_on = CASE WHEN scheduled_on = ? THEN NULL ELSE ? END WHERE id = ?",
		today, today, id,
	)
	if err != nil {
		return fmt.Errorf("toggle today task %d: %w", id, err)
	}
	return nil
}

// SetDueDate sets or clears the due date for a task.
// Pass nil to clear the due date.
func (s *TaskStore) SetDueDate(id int, dueDate *string) error {
	var err error
	if dueDate != nil {
		_, err = s.db.Exec("UPDATE tasks SET due_date = ? WHERE id = ?", *dueDate, id)
	} else {
		_, err = s.db.Exec("UPDATE tasks SET due_date = NULL WHERE id = ?", id)
	}
	if err != nil {
		return fmt.Errorf("set due date task %d: %w", id, err)
	}
	return nil
}

// Delete removes a task by ID. Child tasks are cascade-deleted.
func (s *TaskStore) Delete(id int) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	return nil
}

// HasChildren checks if a task has any child tasks.
func (s *TaskStore) HasChildren(id int) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE parent_id = ?", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check children of task %d: %w", id, err)
	}
	return count > 0, nil
}

// Close closes the database connection.
func (s *TaskStore) Close() error {
	return s.db.Close()
}
