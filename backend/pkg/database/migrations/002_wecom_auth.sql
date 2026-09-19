-- 企业微信账号与平台用户的绑定关系
CREATE TABLE IF NOT EXISTS user_wecom_bindings (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  wecom_userid TEXT NOT NULL UNIQUE,
  bound_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 企业微信认证提供方默认配置行
INSERT INTO auth_provider_settings(id, name, enabled, config)
VALUES ('wecom', '企业微信', FALSE, '{}') ON CONFLICT DO NOTHING;
