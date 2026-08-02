package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*Postgres, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("gong_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	dsn, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	// Create table
	_, err = db.Pool.Exec(ctx, `
		SET TIME ZONE 'UTC';
		CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			target TEXT,
			message TEXT,
			send_at TIMESTAMP,
			status TEXT,
			retry INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	cleanup := func() {
		db.Pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

func newTestNotification() Notification {
	return Notification{
		ID:      uuid.New().String(),
		Target:  "user@example.com",
		Message: "Test notification",
		SendAt:  time.Now().Add(1 * time.Hour),
		Status:  "pending",
	}
}

func TestPostgres_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	n := newTestNotification()

	err := db.Create(ctx, n)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify it was created
	got, err := db.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID() after Create failed: %v", err)
	}

	if got.ID != n.ID {
		t.Errorf("expected ID %q, got %q", n.ID, got.ID)
	}
	if got.Target != n.Target {
		t.Errorf("expected Target %q, got %q", n.Target, got.Target)
	}
	if got.Message != n.Message {
		t.Errorf("expected Message %q, got %q", n.Message, got.Message)
	}
	if got.Status != "pending" {
		t.Errorf("expected Status 'pending', got %q", got.Status)
	}
	if got.Retry != 0 {
		t.Errorf("expected Retry 0, got %d", got.Retry)
	}
}

func TestPostgres_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, err := db.GetByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestPostgres_Cancel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	n := newTestNotification()

	if err := db.Create(ctx, n); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	ok, err := db.Cancel(ctx, n.ID)
	if err != nil {
		t.Fatalf("Cancel() failed: %v", err)
	}
	if !ok {
		t.Error("expected Cancel to return true")
	}

	// Verify status changed
	got, err := db.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID() after Cancel failed: %v", err)
	}
	if got.Status != "canceled" {
		t.Errorf("expected status 'canceled', got %q", got.Status)
	}
}

func TestPostgres_Cancel_AlreadySent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	n := newTestNotification()

	if err := db.Create(ctx, n); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := db.MarkSent(ctx, n.ID); err != nil {
		t.Fatalf("MarkSent() failed: %v", err)
	}

	ok, err := db.Cancel(ctx, n.ID)
	if err != nil {
		t.Fatalf("Cancel() failed: %v", err)
	}
	if ok {
		t.Error("expected Cancel to return false for already sent notification")
	}
}

func TestPostgres_Cancel_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	ok, err := db.Cancel(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Cancel() failed: %v", err)
	}
	if ok {
		t.Error("expected Cancel to return false for non-existent notification")
	}
}

func TestPostgres_MarkSent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	n := newTestNotification()

	if err := db.Create(ctx, n); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := db.MarkSent(ctx, n.ID); err != nil {
		t.Fatalf("MarkSent() failed: %v", err)
	}

	got, err := db.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}
	if got.Status != "sent" {
		t.Errorf("expected status 'sent', got %q", got.Status)
	}
}

func TestPostgres_MarkFailed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	n := newTestNotification()

	if err := db.Create(ctx, n); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := db.MarkFailed(ctx, n.ID); err != nil {
		t.Fatalf("MarkFailed() failed: %v", err)
	}

	got, err := db.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}
	if got.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", got.Status)
	}
}

func TestPostgres_MarkProcessing(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	n := newTestNotification()

	if err := db.Create(ctx, n); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := db.MarkProcessing(ctx, n.ID); err != nil {
		t.Fatalf("MarkProcessing() failed: %v", err)
	}

	got, err := db.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}
	if got.Status != "processing" {
		t.Errorf("expected status 'processing', got %q", got.Status)
	}
}

func TestPostgres_IncrementRetry(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	n := newTestNotification()

	if err := db.Create(ctx, n); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Increment retry 3 times
	for i := 1; i <= 3; i++ {
		if err := db.IncrementRetry(ctx, n.ID); err != nil {
			t.Fatalf("IncrementRetry() attempt %d failed: %v", i, err)
		}

		got, err := db.GetByID(ctx, n.ID)
		if err != nil {
			t.Fatalf("GetByID() after increment %d failed: %v", i, err)
		}
		if got.Retry != i {
			t.Errorf("expected Retry %d, got %d", i, got.Retry)
		}
		if got.Status != "pending" {
			t.Errorf("expected status 'pending' after retry, got %q", got.Status)
		}
	}
}

func TestPostgres_GetPending(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	// Past notification (should be returned)
	past := Notification{
		ID:      uuid.New().String(),
		Target:  "past@example.com",
		Message: "Past notification",
		SendAt:  now.Add(-1 * time.Hour),
		Status:  "pending",
	}

	// Future notification (should NOT be returned)
	future := Notification{
		ID:      uuid.New().String(),
		Target:  "future@example.com",
		Message: "Future notification",
		SendAt:  now.Add(1 * time.Hour),
		Status:  "pending",
	}

	// Already sent notification (should NOT be returned)
	sent := Notification{
		ID:      uuid.New().String(),
		Target:  "sent@example.com",
		Message: "Sent notification",
		SendAt:  now.Add(-1 * time.Hour),
		Status:  "sent",
	}

	for _, n := range []Notification{past, future, sent} {
		n := n
		if err := db.Create(ctx, n); err != nil {
			t.Fatalf("Create(%s) failed: %v", n.ID, err)
		}
	}

	// Mark the sent one
	if err := db.MarkSent(ctx, sent.ID); err != nil {
		t.Fatalf("MarkSent() failed: %v", err)
	}

	pending, err := db.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending() failed: %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending notification, got %d", len(pending))
	}

	if pending[0].ID != past.ID {
		t.Errorf("expected pending ID %q, got %q", past.ID, pending[0].ID)
	}
}

func TestPostgres_StoreInterface(t *testing.T) {
	// Compile-time check that Postgres satisfies Store
	var _ Store = (*Postgres)(nil)
}
