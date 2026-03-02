package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

type Event struct {
	ID         int
	GroupJID   string
	AdminLID   string
	AdminPhone string
	CreatedAt  time.Time
	Active     bool
}

type Participant struct {
	ID           int
	EventID      int
	LID          string
	Phone        string
	Responded    bool
	ResponseText string
	RespondedAt  *time.Time
}

func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Migrate(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			group_jid TEXT NOT NULL,
			admin_lid TEXT NOT NULL,
			admin_phone TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ,
			active BOOLEAN NOT NULL DEFAULT TRUE
		);

		CREATE TABLE IF NOT EXISTS event_participants (
			id SERIAL PRIMARY KEY,
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			lid TEXT NOT NULL,
			phone TEXT NOT NULL,
			responded BOOLEAN NOT NULL DEFAULT FALSE,
			response_text TEXT,
			responded_at TIMESTAMPTZ,
			UNIQUE(event_id, phone)
		);
	`)
	return err
}

func (d *DB) DeactivateGroupEvents(ctx context.Context, groupJID string) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE events SET active = FALSE WHERE group_jid = $1 AND active = TRUE",
		groupJID,
	)
	return err
}

func (d *DB) CreateEvent(ctx context.Context, groupJID, adminLID, adminPhone string) (int, error) {
	var id int
	err := d.pool.QueryRow(ctx,
		"INSERT INTO events (group_jid, admin_lid, admin_phone) VALUES ($1, $2, $3) RETURNING id",
		groupJID, adminLID, adminPhone,
	).Scan(&id)
	return id, err
}

func (d *DB) AddParticipant(ctx context.Context, eventID int, lid, phone string) error {
	_, err := d.pool.Exec(ctx,
		"INSERT INTO event_participants (event_id, lid, phone) VALUES ($1, $2, $3) ON CONFLICT (event_id, phone) DO NOTHING",
		eventID, lid, phone,
	)
	return err
}

func (d *DB) RecordResponse(ctx context.Context, eventID int, phone, responseText string, respondedAt time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx,
		`UPDATE event_participants
		 SET responded = TRUE, response_text = $1, responded_at = $2
		 WHERE event_id = $3 AND phone = $4 AND responded = FALSE`,
		responseText, respondedAt, eventID, phone,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (d *DB) GetActiveEvent(ctx context.Context, groupJID string) (*Event, error) {
	var e Event
	err := d.pool.QueryRow(ctx,
		`SELECT id, group_jid, admin_lid, admin_phone, created_at, active
		 FROM events WHERE group_jid = $1 AND active = TRUE
		 ORDER BY id DESC LIMIT 1`,
		groupJID,
	).Scan(&e.ID, &e.GroupJID, &e.AdminLID, &e.AdminPhone, &e.CreatedAt, &e.Active)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *DB) GetParticipants(ctx context.Context, eventID int) ([]Participant, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT id, event_id, lid, phone, responded, COALESCE(response_text, ''), responded_at
		 FROM event_participants WHERE event_id = $1 ORDER BY id`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []Participant
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.EventID, &p.LID, &p.Phone, &p.Responded, &p.ResponseText, &p.RespondedAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (d *DB) GetUnrespondedCount(ctx context.Context, eventID int) (int, error) {
	var count int
	err := d.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM event_participants WHERE event_id = $1 AND responded = FALSE",
		eventID,
	).Scan(&count)
	return count, err
}

func (d *DB) CompleteEvent(ctx context.Context, eventID int) error {
	_, err := d.pool.Exec(ctx,
		"UPDATE events SET active = FALSE, completed_at = NOW() WHERE id = $1",
		eventID,
	)
	return err
}

func (d *DB) GetAllActiveEvents(ctx context.Context) ([]Event, error) {
	rows, err := d.pool.Query(ctx,
		"SELECT id, group_jid, admin_lid, admin_phone, created_at, active FROM events WHERE active = TRUE",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.GroupJID, &e.AdminLID, &e.AdminPhone, &e.CreatedAt, &e.Active); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (d *DB) Close() {
	d.pool.Close()
}
