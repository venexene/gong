// Package storage provides a PostgreSQL-backed notification store.
package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the minimal interface for notification persistence
type Store interface {
	Create(ctx context.Context, n Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	Cancel(ctx context.Context, id string) (bool, error)
}

// Postgres implements Store using a pgx connection pool.
type Postgres struct {
	Pool *pgxpool.Pool
}

// Compile-time check that Postgres satisfies Store.
var _ Store = (*Postgres)(nil)

// New creates a new Postgres store connected to the given DSN.
func New(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &Postgres{Pool: pool}, nil
}

// Notification represents a delayed notification stored in the database.
type Notification struct {
	ID        string    // Unique identifier.
	Target    string    // Recipient address.
	Message   string    // Notification body.
	SendAt    time.Time // Scheduled delivery time.
	Status    string    // Current state: pending, processing, sent, canceled, failed.
	Retry     int       // Number of delivery attempts.
	CreatedAt time.Time // Timestamp of creation.
}

// Create inserts a new notification into the database.
func (p *Postgres) Create(ctx context.Context, n Notification) error {
	query := `
        INSERT INTO notifications (id, target, message, send_at, status)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := p.Pool.Exec(ctx, query,
		n.ID, n.Target, n.Message, n.SendAt, n.Status,
	)
	return err
}

// GetByID retrieves a single notification by its ID.
func (p *Postgres) GetByID(ctx context.Context, id string) (*Notification, error) {
	query := `
        SELECT id, target, message, send_at, status, retry, created_at
        FROM notifications
        WHERE id = $1
    `

	var n Notification

	err := p.Pool.QueryRow(ctx, query, id).Scan(
		&n.ID,
		&n.Target,
		&n.Message,
		&n.SendAt,
		&n.Status,
		&n.Retry,
		&n.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &n, nil
}

// Cancel marks a pending notification as canceled.
func (p *Postgres) Cancel(ctx context.Context, id string) (bool, error) {
	query := `
		UPDATE notifications
		SET status = 'canceled'
		WHERE id = $1 AND status = 'pending'
	`
	res, err := p.Pool.Exec(ctx, query, id)
	if err != nil {
		return false, err
	}

	if res.RowsAffected() == 0 {
		return false, nil
	}

	return true, nil
}

// GetPending returns all notifications that are pending and whose send_at time has already passed
func (p *Postgres) GetPending(ctx context.Context) ([]Notification, error) {
	rows, err := p.Pool.Query(ctx, `
        SELECT id, target, message, send_at, status, retry
        FROM notifications
        WHERE status = 'pending' AND send_at <= NOW()
        ORDER BY send_at ASC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Target, &n.Message, &n.SendAt, &n.Status, &n.Retry); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, nil
}

// MarkSent sets the notification status to "sent".
func (p *Postgres) MarkSent(ctx context.Context, id string) error {
	_, err := p.Pool.Exec(ctx, `UPDATE notifications SET status='sent' WHERE id=$1`, id)
	return err
}

// IncrementRetry bumps the retry counter and resets status to "pending"
func (p *Postgres) IncrementRetry(ctx context.Context, id string) error {
	_, err := p.Pool.Exec(ctx, `
        UPDATE notifications 
        SET retry = retry + 1, status = 'pending'
        WHERE id = $1
    `, id)
	return err
}

// MarkFailed sets the notification status to "failed" after exhausting all retry attempts.
func (p *Postgres) MarkFailed(ctx context.Context, id string) error {
	_, err := p.Pool.Exec(ctx,
		`UPDATE notifications
         SET status='failed'
          WHERE id=$1`, id)
	return err
}

// MarkProcessing atomically transitions a pending notification to "processing" to prevent concurrent delivery.
func (p *Postgres) MarkProcessing(ctx context.Context, id string) error {
	_, err := p.Pool.Exec(ctx, `
        UPDATE notifications
        SET status='processing'
        WHERE id=$1 AND status='pending'
    `, id)
	return err
}
