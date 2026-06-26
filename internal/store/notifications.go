package store

import (
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateNotification(n *models.Notification) error {
	res, err := s.db.Exec(
		`INSERT INTO notifications (user_id, message, link) VALUES (?, ?, ?)`,
		n.UserID, n.Message, n.Link,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	n.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) ListNotificationsByUser(userID int64) ([]*models.Notification, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, message, link, read, created_at FROM notifications WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
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

func (s *Store) MarkNotificationRead(id, userID int64) error {
	res, err := s.db.Exec(`UPDATE notifications SET read = 1 WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

func (s *Store) CountUnreadNotifications(userID int64) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read = 0`, userID).Scan(&count)
	return count, err
}
