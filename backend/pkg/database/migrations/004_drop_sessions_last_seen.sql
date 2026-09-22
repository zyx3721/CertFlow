-- 移除会话表的闲置续期列，会话有效期仅由 expires_at 决定
ALTER TABLE sessions DROP COLUMN IF EXISTS last_seen_at;
