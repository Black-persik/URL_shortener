package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"urlShort/internal/service"
)

type LinksRepo struct {
	db *sql.DB
}

func NewLinksRepo(db *sql.DB) *LinksRepo {
	return &LinksRepo{db: db}
}

func (r *LinksRepo) CreateLink(ctx context.Context, code, originalURL string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO links(code, original_url) VALUES ($1, $2)`,
		code, originalURL,
	)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 = unique_violation
		if pgErr.Code == "23505" {
			return service.ErrConflict
		}
	}
	return err
}

func (r *LinksRepo) GetLinkByCode(ctx context.Context, code string) (id int64, originalURL string, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, original_url FROM links WHERE code=$1`,
		code,
	)
	if err := row.Scan(&id, &originalURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", service.ErrNotFound
		}
		return 0, "", err
	}
	return id, originalURL, nil
}

func (r *LinksRepo) InsertClicks(ctx context.Context, events []service.ClickEvent) error {
	if len(events) == 0 {
		return nil
	}

	// multi-row insert: VALUES ($1,$2,$3,$4), ($5,$6,$7,$8) ...
	var b strings.Builder
	b.WriteString(`INSERT INTO clicks(link_id, ts, ip, user_agent) VALUES `)

	args := make([]any, 0, len(events)*4)
	for i, ev := range events {
		if i > 0 {
			b.WriteString(",")
		}
		n := i*4 + 1
		fmt.Fprintf(&b, " ($%d,$%d,$%d,$%d)", n, n+1, n+2, n+3)

		args = append(args, ev.LinkID, ev.TS, ev.IP, ev.UserAgent)
	}

	_, err := r.db.ExecContext(ctx, b.String(), args...)
	return err
}

func (r *LinksRepo) TotalClicks(ctx context.Context, code string) (int64, error) {
	// Если ссылки нет — вернём ErrNotFound (потому что запроса не к чему применить)
	row := r.db.QueryRowContext(ctx, `
		SELECT COUNT(c.id)
		FROM links l
		LEFT JOIN clicks c ON c.link_id = l.id
		WHERE l.code = $1
		GROUP BY l.id
	`, code)

	var cnt int64
	if err := row.Scan(&cnt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, service.ErrNotFound
		}
		return 0, err
	}
	return cnt, nil
}
