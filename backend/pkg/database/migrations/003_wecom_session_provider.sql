-- 会话认证来源扩展企业微信扫码登录
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_auth_provider_check;
ALTER TABLE sessions ADD CONSTRAINT sessions_auth_provider_check CHECK (auth_provider IN ('local', 'ldap', 'wecom'));
