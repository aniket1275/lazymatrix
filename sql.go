package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	_ "github.com/mattn/go-sqlite3"
)

type Note struct {
	ID      int64
	Title   string
	Body    string
	Section string
}

type Store struct {
	conn *sql.DB
}

func (s *Store) Init() error {
	var err error
	dbPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "lazytime")
	dbFile := filepath.Join(dbPath, "lists.db")

	// Create directory if it doesn't exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		err := os.MkdirAll(dbPath, 0755)
		if err != nil {
			return err
		}
	}
	s.conn, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}
	createTableStmt := `
		CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL UNIQUE,
		body TEXT NOT NULL,
		section TEXT NOT NULL
	);`
	if _, err = s.conn.Exec(createTableStmt); err != nil {
		return err
	}
	return nil
}

func FetchItemsBySection(db *sql.DB, section string) []list.Item {
	rows, err := db.Query("SELECT title, body FROM notes WHERE section = ?", section)
	if err != nil {
		log.Println("Query error:", err)
		return nil
	}
	defer rows.Close()

	var items []list.Item
	for rows.Next() {
		var title, body string
		if err := rows.Scan(&title, &body); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		items = append(items, item{title: title, desc: body})
	}

	return items
}

// func (s *Store) Seed() {
// 	stmt := `INSERT INTO notes (title, body, section) VALUES (?, ?, ?)`
// 	notes := []struct {
// 		title   string
// 		body    string
// 		section string
// 	}{
// 		{"Linux", "Open-source OS", "A"},
// 		{"Go", "Fast, compiled language", "A"},
// 		{"Rust", "Memory-safe systems language", "A"},
// 		{"Neovim", "Extensible Vim-based editor", "B"},
// 		{"Emacs", "It's like an OS", "B"},
// 		{"VS Code", "Most popular editor", "B"},
// 		{"Git", "Distributed version control", "C"},
// 		{"Docker", "Containerization platform", "C"},
// 		{"Kubernetes", "Orchestration for containers", "C"},
// 	}
//
// 	for _, note := range notes {
// 		_, err := s.conn.Exec(stmt, note.title, note.body, note.section)
// 		if err != nil {
// 			log.Printf("Insert error for %q: %v\n", note.title, err)
// 		}
// 	}
// }

func (s *Store) Save(title, desc, section string) error {
	saveQuery := `INSERT INTO notes (title, body, section) VALUES (?, ?, ?)`
	_, err := s.conn.Exec(saveQuery, title, desc, section)
	return err
}

func (s *Store) Delete(title, section string) error {
	query := `DELETE FROM notes WHERE title = ? AND section = ?`
	_, err := s.conn.Exec(query, title, section)
	return err
}

func (s *Store) Update(title, desc, section, oldTitle string) error {
	query := `UPDATE notes SET title = ?, body = ?, section = ? WHERE title = ?`
	_, err := s.conn.Exec(query, title, desc, section, oldTitle)
	return err
}
