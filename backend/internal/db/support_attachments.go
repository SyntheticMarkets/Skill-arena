package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"skill-arena/internal/id"
	"skill-arena/internal/models"
)

func (s *Store) StoreSupportAttachment(ctx context.Context, userID, ticketID, fileName, contentType string, data []byte) (*models.SupportAttachment, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if userID == "" || ticketID == "" || fileName == "" || len(fileName) > 255 {
		return nil, errors.New("attachment owner, ticket, and file name are required")
	}
	if contentType != "application/pdf" && contentType != "image/jpeg" && contentType != "image/png" && contentType != "text/plain" {
		return nil, errors.New("support attachments must be PDF, JPEG, PNG, or plain text")
	}
	if len(data) == 0 || len(data) > 10<<20 {
		return nil, errors.New("support attachments must be between 1 byte and 10 MiB")
	}
	tickets, err := s.ListSupportTickets(ctx, userID)
	if err != nil {
		return nil, err
	}
	owned := false
	for _, ticket := range tickets {
		if ticket.ID == ticketID {
			owned = true
			break
		}
	}
	if !owned {
		return nil, errors.New("support ticket was not found")
	}
	now := time.Now().UTC()
	item := &models.SupportAttachment{
		ID: id.New("support-file"), TicketID: ticketID, UserID: userID, FileName: fileName,
		ContentType: contentType, SizeBytes: int64(len(data)), CreatedAt: now,
	}
	item.ObjectKey = fmt.Sprintf("support/%s/%s/%s", userID, ticketID, item.ID)
	digest := sha256.Sum256(data)
	item.SHA256 = hex.EncodeToString(digest[:])
	if err := s.objects.Put(ctx, item.ObjectKey, data, item.ContentType); err != nil {
		return nil, err
	}
	if s.usesPostgresAuth() {
		_, err := s.pg.ExecContext(ctx, `INSERT INTO crm_support_attachments
			(id,ticket_id,user_id,object_key,file_name,content_type,size_bytes,sha256,created_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			item.ID, item.TicketID, item.UserID, item.ObjectKey, item.FileName, item.ContentType,
			item.SizeBytes, item.SHA256, item.CreatedAt)
		if err != nil {
			_ = s.objects.Delete(ctx, item.ObjectKey)
			return nil, err
		}
		return item, nil
	}
	return nil, errors.New("support attachment persistence requires PostgreSQL")
}

func (s *Store) ListSupportAttachments(ctx context.Context, ticketID string) ([]models.SupportAttachment, error) {
	if !s.usesPostgresAuth() {
		return []models.SupportAttachment{}, nil
	}
	rows, err := s.pg.QueryContext(ctx, `SELECT id,ticket_id,user_id,object_key,file_name,content_type,size_bytes,sha256,created_at
		FROM crm_support_attachments WHERE ticket_id=$1 ORDER BY created_at`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.SupportAttachment{}
	for rows.Next() {
		var item models.SupportAttachment
		if err := rows.Scan(&item.ID, &item.TicketID, &item.UserID, &item.ObjectKey, &item.FileName,
			&item.ContentType, &item.SizeBytes, &item.SHA256, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetSupportAttachment(ctx context.Context, attachmentID string) (*models.SupportAttachment, []byte, error) {
	if !s.usesPostgresAuth() {
		return nil, nil, errors.New("support attachment not found")
	}
	var item models.SupportAttachment
	err := s.pg.QueryRowContext(ctx, `SELECT id,ticket_id,user_id,object_key,file_name,content_type,size_bytes,sha256,created_at
		FROM crm_support_attachments WHERE id=$1`, attachmentID).Scan(
		&item.ID, &item.TicketID, &item.UserID, &item.ObjectKey, &item.FileName,
		&item.ContentType, &item.SizeBytes, &item.SHA256, &item.CreatedAt)
	if err != nil {
		return nil, nil, err
	}
	data, err := s.objects.Get(ctx, item.ObjectKey)
	if err != nil {
		return nil, nil, err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != item.SHA256 {
		return nil, nil, errors.New("support attachment integrity check failed")
	}
	return &item, data, nil
}
