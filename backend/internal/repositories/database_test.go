package repositories_test

import (
	"testing"

	"task-manager/backend/internal/repositories"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)

	db.Exec(`CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		due_date DATETIME,
		priority TEXT NOT NULL DEFAULT 'Low',
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)

	db.Exec(`CREATE TABLE tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	return db, nil
}

func TestDatabaseConfig_Creation(t *testing.T) {
	config := repositories.NewDatabaseConfig()

	if config == nil {
		t.Error("Expected non-nil database config")
	}

	if config.Host == "" {
		t.Error("Expected non-empty host")
	}

	if config.Port == "" {
		t.Error("Expected non-empty port")
	}
}

func TestDatabaseConnection_Ping(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestDatabaseTables_Existence(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	tables := []string{"users", "tasks", "tokens"}

	for _, table := range tables {
		var count int64
		err := db.Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error
		if err != nil {
			t.Errorf("Failed to query table %s: %v", table, err)
		} else {
			t.Logf("Table %s exists and is queryable", table)
		}
	}
}

func TestDatabase_BasicOperations(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	err = db.Exec("INSERT INTO users (id, username, password) VALUES (?, ?, ?)",
		"test-id-1", "testuser", "hashedpassword").Error
	if err != nil {
		t.Errorf("Failed to insert test user: %v", err)
	}

	var username string
	err = db.Raw("SELECT username FROM users WHERE id = ?", "test-id-1").Scan(&username).Error
	if err != nil {
		t.Errorf("Failed to read test user: %v", err)
	}

	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", username)
	}

	err = db.Exec("UPDATE users SET username = ? WHERE id = ?", "updateduser", "test-id-1").Error
	if err != nil {
		t.Errorf("Failed to update test user: %v", err)
	}

	err = db.Raw("SELECT username FROM users WHERE id = ?", "test-id-1").Scan(&username).Error
	if err != nil {
		t.Errorf("Failed to read updated user: %v", err)
	}

	if username != "updateduser" {
		t.Errorf("Expected username 'updateduser', got '%s'", username)
	}

	err = db.Exec("DELETE FROM users WHERE id = ?", "test-id-1").Error
	if err != nil {
		t.Errorf("Failed to delete test user: %v", err)
	}

	var count int64
	err = db.Raw("SELECT COUNT(*) FROM users WHERE id = ?", "test-id-1").Scan(&count).Error
	if err != nil {
		t.Errorf("Failed to count users after deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 users after deletion, got %d", count)
	}
}

func TestDatabase_Transactions(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	tx := db.Begin()

	err = tx.Exec("INSERT INTO users (id, username, password) VALUES (?, ?, ?)",
		"tx-test-1", "txuser", "password").Error
	if err != nil {
		t.Errorf("Failed to insert in transaction: %v", err)
	}

	tx.Rollback()

	var count int64
	err = db.Raw("SELECT COUNT(*) FROM users WHERE id = ?", "tx-test-1").Scan(&count).Error
	if err != nil {
		t.Errorf("Failed to count users after rollback: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 users after rollback, got %d", count)
	}

	tx = db.Begin()

	err = tx.Exec("INSERT INTO users (id, username, password) VALUES (?, ?, ?)",
		"tx-test-2", "txuser2", "password").Error
	if err != nil {
		t.Errorf("Failed to insert in transaction: %v", err)
	}

	tx.Commit()

	err = db.Raw("SELECT COUNT(*) FROM users WHERE id = ?", "tx-test-2").Scan(&count).Error
	if err != nil {
		t.Errorf("Failed to count users after commit: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user after commit, got %d", count)
	}
}
