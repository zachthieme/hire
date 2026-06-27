package store

import (
	"context"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateNotification(ctx context.Context, n *models.Notification) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO notifications (user_id, message, link) VALUES ($1, $2, $3) RETURNING id`,
		n.UserID, n.Message, n.Link,
	).Scan(&n.ID)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (s *Store) ListNotificationsByUser(ctx context.Context, userID int64, limit, offset int) ([]*models.Notification, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, message, link, is_read, created_at FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Message, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (s *Store) MarkNotificationRead(ctx context.Context, id, userID int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CountUnreadNotifications(ctx context.Context, userID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&count)
	return count, err
}
