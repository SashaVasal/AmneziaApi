package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"amnezia_go/backend/internal/models"
	_ "modernc.org/sqlite"
)

var ErrPeerNotFound = errors.New("peer not found")

type PeerRepository interface {
	Init(ctx context.Context) error
	ListPeers(ctx context.Context) ([]models.Peer, error)
	GetPeerByID(ctx context.Context, id string) (models.Peer, error)
	CreatePeer(ctx context.Context, peer models.Peer) error
	DeletePeer(ctx context.Context, id string) error
	ListAllocatedIPs(ctx context.Context) ([]string, error)
}

type SQLitePeerRepository struct {
	db *sql.DB
}

func NewSQLitePeerRepository(dbPath string) (*SQLitePeerRepository, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	return &SQLitePeerRepository{db: db}, nil
}

func (r *SQLitePeerRepository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *SQLitePeerRepository) Init(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS peers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	public_key TEXT NOT NULL UNIQUE,
	private_key TEXT NOT NULL,
	allowed_ip TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL
);
`)
	if err != nil {
		return fmt.Errorf("create peers table: %w", err)
	}
	return nil
}

func (r *SQLitePeerRepository) ListPeers(ctx context.Context) ([]models.Peer, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, public_key, private_key, allowed_ip, created_at
FROM peers
ORDER BY created_at DESC
`)
	if err != nil {
		return nil, fmt.Errorf("list peers: %w", err)
	}
	defer rows.Close()

	peers := make([]models.Peer, 0)
	for rows.Next() {
		peer, err := scanPeerRow(rows)
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate peers: %w", err)
	}
	return peers, nil
}

func (r *SQLitePeerRepository) GetPeerByID(ctx context.Context, id string) (models.Peer, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, public_key, private_key, allowed_ip, created_at
FROM peers
WHERE id = ?
`, id)

	peer, err := scanPeerRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Peer{}, ErrPeerNotFound
		}
		return models.Peer{}, err
	}
	return peer, nil
}

func (r *SQLitePeerRepository) CreatePeer(ctx context.Context, peer models.Peer) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO peers(id, name, public_key, private_key, allowed_ip, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`, peer.ID, peer.Name, peer.PublicKey, peer.PrivateKey, peer.AllowedIP, peer.CreatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert peer: %w", err)
	}
	return nil
}

func (r *SQLitePeerRepository) DeletePeer(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM peers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete peer: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrPeerNotFound
	}
	return nil
}

func (r *SQLitePeerRepository) ListAllocatedIPs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT allowed_ip FROM peers`)
	if err != nil {
		return nil, fmt.Errorf("list allocated ips: %w", err)
	}
	defer rows.Close()

	ips := make([]string, 0)
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("scan allocated ip: %w", err)
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPeerRow(s scanner) (models.Peer, error) {
	var peer models.Peer
	var createdAtRaw string

	if err := s.Scan(
		&peer.ID,
		&peer.Name,
		&peer.PublicKey,
		&peer.PrivateKey,
		&peer.AllowedIP,
		&createdAtRaw,
	); err != nil {
		return models.Peer{}, err
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return models.Peer{}, fmt.Errorf("parse created_at: %w", err)
	}
	peer.CreatedAt = createdAt.UTC()
	return peer, nil
}
