package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	return &TaskStore{db: db}, nil
}

// Add inserts a new task and returns it.
func (s *TaskStore) Add(title string) (Task, error) {
	res, err := s.db.Exec("INSERT INTO tasks (title) VALUES (?)", title)
	if err != nil {
		return Task{}, fmt.Errorf("insert task: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(int(id))
}

// List returns all tasks ordered by creation date descending.
func (s *TaskStore) List() ([]Task, error) {
	rows, err := s.db.Query("SELECT id, title, completed, created_at FROM tasks ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var comp int
		var createdStr string
		if err := rows.Scan(&t.ID, &t.Title, &comp, &createdStr); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Completed = comp != 0
		t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetByID retrieves a single task by its ID.
func (s *TaskStore) GetByID(id int) (Task, error) {
	var t Task
	var comp int
	var createdStr string
	err := s.db.QueryRow("SELECT id, title, completed, created_at FROM tasks WHERE id = ?", id).
		Scan(&t.ID, &t.Title, &comp, &createdStr)
	if err != nil {
		return Task{}, fmt.Errorf("get task %d: %w", id, err)
	}
	t.Completed = comp != 0
	t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
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

// Delete removes a task by ID.
func (s *TaskStore) Delete(id int) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	return nil
}

// Close closes the database connection.
func (s *TaskStore) Close() error {
	return s.db.Close()
}
