package repository

import (
	"context"
	"encoding/json"
)

func (s *Store) Settings(ctx context.Context) (map[string]any, error) {
	var raw []byte
	if err := s.Pool.QueryRow(ctx, "SELECT value FROM system_settings WHERE key = 'pki'").Scan(&raw); err != nil {
		return nil, err
	}

	settings := map[string]any{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *Store) SaveSettings(ctx context.Context, settings map[string]any) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO system_settings(key, value)
		VALUES ('pki', $1)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now()
	`, raw)
	return err
}
