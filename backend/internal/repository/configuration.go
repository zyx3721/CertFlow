package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type ExpiringCertificate struct {
	ID         string
	CommonName string
	NotAfter   time.Time
}

func (s *Store) ListUnnotifiedExpiringCertificates(ctx context.Context, until time.Time, channelID string) ([]ExpiringCertificate, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT c.id::text,c.common_name,c.not_after
		FROM certificates c
		WHERE c.status='valid' AND c.not_after>now() AND c.not_after<=$1
		  AND NOT EXISTS (
			SELECT 1 FROM certificate_expiry_notifications n
			WHERE n.certificate_id=c.id AND n.notice_date=CURRENT_DATE AND n.channel_id=$2
		  )
		ORDER BY c.not_after,c.common_name
	`, until, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ExpiringCertificate{}
	for rows.Next() {
		var item ExpiringCertificate
		if err := rows.Scan(&item.ID, &item.CommonName, &item.NotAfter); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) MarkCertificateExpiryNotified(ctx context.Context, certificateID, channelID string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO certificate_expiry_notifications(certificate_id,notice_date,channel_id)
		VALUES($1,CURRENT_DATE,$2)
		ON CONFLICT(certificate_id,notice_date,channel_id) DO NOTHING
	`, certificateID, channelID)
	return err
}

func (s *Store) ClearCertificateExpiryNotificationsForChannel(ctx context.Context, channelID string) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM certificate_expiry_notifications WHERE channel_id=$1", channelID)
	return err
}

type AuthProviderSetting struct {
	ID               string         `json:"id"`
	Type             string         `json:"type"`
	Name             string         `json:"name"`
	Enabled          bool           `json:"enabled"`
	Config           map[string]any `json:"config"`
	SecretCiphertext string         `json:"-"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type NotificationChannelSetting struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	PasswordResetEnabled bool           `json:"passwordResetEnabled"`
	ApprovalEnabled      bool           `json:"approvalEnabled"`
	Config               map[string]any `json:"config"`
	SecretCiphertext     string         `json:"-"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

func (s *Store) AuthProviderSetting(ctx context.Context, id string) (AuthProviderSetting, error) {
	var item AuthProviderSetting
	var raw []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id,name,enabled,config,secret_ciphertext,updated_at
		FROM auth_provider_settings WHERE id=$1
	`, id).Scan(&item.ID, &item.Name, &item.Enabled, &raw, &item.SecretCiphertext, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.Type = id
	item.Config = map[string]any{}
	if err := json.Unmarshal(raw, &item.Config); err != nil {
		return item, err
	}
	return item, nil
}

func (s *Store) SaveAuthProviderSetting(ctx context.Context, item AuthProviderSetting) error {
	raw, err := json.Marshal(item.Config)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO auth_provider_settings(id,name,enabled,config,secret_ciphertext)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,enabled=EXCLUDED.enabled,
		  config=EXCLUDED.config,secret_ciphertext=EXCLUDED.secret_ciphertext,updated_at=now()
	`, item.ID, item.Name, item.Enabled, raw, item.SecretCiphertext)
	return err
}

func (s *Store) NotificationChannelSetting(ctx context.Context, id string) (NotificationChannelSetting, error) {
	var item NotificationChannelSetting
	var raw []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id,name,password_reset_enabled,approval_enabled,config,secret_ciphertext,updated_at
		FROM notification_channel_settings WHERE id=$1
	`, id).Scan(&item.ID, &item.Name, &item.PasswordResetEnabled, &item.ApprovalEnabled, &raw, &item.SecretCiphertext, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.Config = map[string]any{}
	if err := json.Unmarshal(raw, &item.Config); err != nil {
		return item, err
	}
	return item, nil
}

func (s *Store) ListNotificationChannelSettings(ctx context.Context) ([]NotificationChannelSetting, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id,name,password_reset_enabled,approval_enabled,config,secret_ciphertext,updated_at
		FROM notification_channel_settings
		ORDER BY CASE id
			WHEN 'webhook' THEN 1 WHEN 'email' THEN 2 WHEN 'lark' THEN 3 WHEN 'lark_app' THEN 4
			WHEN 'wechat' THEN 5 WHEN 'wechat_app' THEN 6 WHEN 'dingtalk' THEN 7 WHEN 'dingtalk_app' THEN 8
			ELSE 9 END, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []NotificationChannelSetting{}
	for rows.Next() {
		var item NotificationChannelSetting
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.PasswordResetEnabled, &item.ApprovalEnabled, &raw, &item.SecretCiphertext, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Config = map[string]any{}
		if err := json.Unmarshal(raw, &item.Config); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SaveNotificationChannelSetting(ctx context.Context, item NotificationChannelSetting) error {
	raw, err := json.Marshal(item.Config)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO notification_channel_settings(id,name,password_reset_enabled,approval_enabled,config,secret_ciphertext)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,password_reset_enabled=EXCLUDED.password_reset_enabled,approval_enabled=EXCLUDED.approval_enabled,
		  config=EXCLUDED.config,secret_ciphertext=EXCLUDED.secret_ciphertext,updated_at=now()
	`, item.ID, item.Name, item.PasswordResetEnabled, item.ApprovalEnabled, raw, item.SecretCiphertext)
	return err
}
