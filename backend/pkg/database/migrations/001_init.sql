-- 平台用户账户
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  email TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT 'viewer',
  source TEXT NOT NULL CHECK (source IN ('local', 'ldap')) DEFAULT 'local',
  disabled BOOLEAN NOT NULL DEFAULT FALSE,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 用户登录会话
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  auth_provider TEXT NOT NULL DEFAULT 'local' CHECK (auth_provider IN ('local', 'ldap')),
  expires_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- X.509 证书颁发机构
CREATE TABLE IF NOT EXISTS certificate_authorities (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  type TEXT NOT NULL CHECK (type IN ('root', 'intermediate', 'issuing')),
  algorithm TEXT NOT NULL,
  subject TEXT NOT NULL,
  parent_id UUID REFERENCES certificate_authorities(id) ON DELETE CASCADE,
  certificate_pem TEXT NOT NULL,
  private_key_ciphertext TEXT NOT NULL,
  not_before TIMESTAMPTZ NOT NULL,
  not_after TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  issued_certs INTEGER NOT NULL DEFAULT 0,
  revoked_certs INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 为每个签发 CA 隔离的 OCSP 响应签名身份
CREATE TABLE IF NOT EXISTS ocsp_responders (
  id UUID PRIMARY KEY,
  issuer_ca_id UUID NOT NULL UNIQUE REFERENCES certificate_authorities(id) ON DELETE CASCADE,
  certificate_pem TEXT NOT NULL,
  private_key_ciphertext TEXT NOT NULL,
  not_before TIMESTAMPTZ NOT NULL,
  not_after TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 平台签发和申请的证书
CREATE TABLE IF NOT EXISTS certificates (
  id UUID PRIMARY KEY,
  ca_id UUID NOT NULL REFERENCES certificate_authorities(id) ON DELETE RESTRICT,
  serial_number TEXT NOT NULL UNIQUE,
  common_name TEXT NOT NULL,
  subject TEXT NOT NULL,
  san JSONB NOT NULL DEFAULT '[]',
  algorithm TEXT NOT NULL,
  -- 证书用途支持服务器、客户端与 mTLS 双向认证
  purpose TEXT NOT NULL CHECK (purpose IN ('server', 'client', 'mtls')),
  validity_days INTEGER NOT NULL DEFAULT 90,
  status TEXT NOT NULL CHECK (status IN ('pending', 'valid', 'revoked', 'rejected', 'expired')) DEFAULT 'pending',
  applicant_id UUID REFERENCES users(id) ON DELETE SET NULL,
  applicant TEXT NOT NULL DEFAULT '',
  csr_pem TEXT NOT NULL DEFAULT '',
  certificate_pem TEXT NOT NULL DEFAULT '',
  private_key_ciphertext TEXT NOT NULL DEFAULT '',
  not_before TIMESTAMPTZ,
  not_after TIMESTAMPTZ,
  reject_reason TEXT NOT NULL DEFAULT '',
  revoked_at TIMESTAMPTZ,
  revoke_reason TEXT NOT NULL DEFAULT '',
  renewed_from_id UUID REFERENCES certificates(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 删除证书后仍保留的独立证书撤销事件
CREATE TABLE IF NOT EXISTS certificate_revocation_entries (
  id UUID PRIMARY KEY,
  certificate_id UUID UNIQUE REFERENCES certificates(id) ON DELETE SET NULL,
  ca_id UUID NOT NULL REFERENCES certificate_authorities(id) ON DELETE RESTRICT,
  applicant_id UUID REFERENCES users(id) ON DELETE SET NULL,
  serial_number TEXT NOT NULL,
  common_name TEXT NOT NULL,
  revoked_at TIMESTAMPTZ NOT NULL,
  revoke_reason TEXT NOT NULL CHECK (revoke_reason IN ('keyCompromise', 'cACompromise', 'affiliationChanged', 'superseded', 'cessationOfOperation', 'unspecified')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_certificate_revocation_entries_ca_revoked_at
  ON certificate_revocation_entries(ca_id, revoked_at DESC);
CREATE INDEX IF NOT EXISTS idx_certificate_revocation_entries_applicant_revoked_at
  ON certificate_revocation_entries(applicant_id, revoked_at DESC);

-- 证书申请审批流程
CREATE TABLE IF NOT EXISTS workflows (
  id UUID PRIMARY KEY,
  certificate_id UUID NOT NULL UNIQUE REFERENCES certificates(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')) DEFAULT 'pending',
  reject_reason TEXT NOT NULL DEFAULT '',
  reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 安全与操作审计记录
CREATE TABLE IF NOT EXISTS audit_entries (
  id UUID PRIMARY KEY,
  ts TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  username TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  target TEXT NOT NULL,
  module TEXT NOT NULL,
  result TEXT NOT NULL,
  ip_address TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT ''
);

-- 平台运行与基础配置
CREATE TABLE IF NOT EXISTS system_settings (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_certificates_status ON certificates(status);
CREATE INDEX IF NOT EXISTS idx_certificates_expiry ON certificates(not_after);
CREATE UNIQUE INDEX IF NOT EXISTS idx_certificates_renewed_from_id
  ON certificates(renewed_from_id) WHERE renewed_from_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_entries_ts ON audit_entries(ts DESC);

INSERT INTO system_settings(key, value)
VALUES (
  'pki',
  '{
    "siteName": "CertFlow",
    "loginName": "CertFlow",
    "appName": "CertFlow",
    "appSubtitle": "PKI Control Plane",
    "iconData": "/favicon.svg",
    "crlEnabled": true,
    "crlUrl": "/crl",
    "ocspEnabled": true,
    "ocspUrl": "/ocsp",
    "crlIntervalHours": 24,
    "renewDays": 10,
    "autoRenew": false,
    "expiryNotificationDays": 15,
    "expiryNotificationCron": "0 0 * * *",
    "resetCodeTtlMinutes": 10,
    "resetCaptchaTtlMinutes": 1,
    "passwordResetSendCooldownMinutes": 0.5,
    "passwordResetRateLimitMinutes": 5
  }'::jsonb
)
ON CONFLICT DO NOTHING;

-- 平台角色与权限集合
CREATE TABLE IF NOT EXISTS roles (
  id UUID PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  permissions TEXT[] NOT NULL DEFAULT '{}',
  builtin BOOLEAN NOT NULL DEFAULT FALSE,
  disabled BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 用户与角色的多对多关系
CREATE TABLE IF NOT EXISTS user_roles (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, role_id)
);

-- 用户群组
CREATE TABLE IF NOT EXISTS user_groups (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  disabled BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 用户群组成员关系
CREATE TABLE IF NOT EXISTS user_group_members (
  group_id UUID NOT NULL REFERENCES user_groups(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY (group_id, user_id)
);

-- 用户群组与角色关系
CREATE TABLE IF NOT EXISTS user_group_roles (
  group_id UUID NOT NULL REFERENCES user_groups(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (group_id, role_id)
);

-- 外部认证提供方配置
CREATE TABLE IF NOT EXISTS auth_provider_settings (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  config JSONB NOT NULL DEFAULT '{}',
  secret_ciphertext TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 通知媒介与找回密码邮件配置
CREATE TABLE IF NOT EXISTS notification_channel_settings (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  password_reset_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  approval_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  config JSONB NOT NULL DEFAULT '{}',
  secret_ciphertext TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 证书到期提醒按媒介的每日发送记录
CREATE TABLE IF NOT EXISTS certificate_expiry_notifications (
  certificate_id UUID NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,
  notice_date DATE NOT NULL,
  channel_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (certificate_id, notice_date, channel_id)
);

-- 找回密码验证请求与邮件验证码
CREATE TABLE IF NOT EXISTS password_reset_requests (
  token_hash TEXT PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL DEFAULT '',
  code_expires_at TIMESTAMPTZ,
  code_sent_at TIMESTAMPTZ,
  used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_requests_user ON password_reset_requests(user_id, created_at DESC);

INSERT INTO roles(id, key, name, description, permissions, builtin)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'admin', 'admin', '系统配置、CA、证书与审批等所有操作', ARRAY[
    'dashboard.read',
    'ca.read', 'ca.download', 'ca.add', 'ca.delete',
    'certificates.read', 'certificates.request', 'certificates.download', 'certificates.verify',
    'certificates.manage', 'certificates.revoke', 'certificates.delete',
    'workflows.read', 'workflows.approve', 'workflows.delete',
    'crl.read', 'crl.manage', 'audit.read', 'audit.manage', 'settings.base.read',
    'settings.base.manage', 'settings.users.read', 'settings.users.manage', 'settings.auth.read',
    'settings.auth.manage', 'settings.notifications.read', 'settings.notifications.manage'
  ], TRUE),
  ('00000000-0000-0000-0000-000000000002', 'operator', 'operator', 'CA、证书与审批日常操作，不能修改系统配置', ARRAY[
    'dashboard.read',
    'ca.read', 'ca.download', 'ca.add', 'ca.delete',
    'certificates.read', 'certificates.request', 'certificates.download', 'certificates.verify',
    'certificates.manage', 'certificates.revoke', 'certificates.delete',
    'workflows.read', 'workflows.approve', 'workflows.delete',
    'crl.read', 'crl.manage', 'audit.read', 'audit.manage'
  ], TRUE),
  ('00000000-0000-0000-0000-000000000003', 'viewer', 'viewer', '只读查看仪表板、CA、证书、CRL 和审计记录', ARRAY[
    'dashboard.read', 'ca.read', 'certificates.read', 'crl.read', 'audit.read'
  ], TRUE)
ON CONFLICT (id) DO UPDATE SET
  key = EXCLUDED.key,
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  permissions = EXCLUDED.permissions,
  builtin = TRUE,
  updated_at = now();

INSERT INTO user_roles(user_id, role_id)
SELECT users.id, roles.id FROM users JOIN roles ON roles.key = users.role
ON CONFLICT DO NOTHING;

INSERT INTO auth_provider_settings(id, name, enabled, config)
VALUES ('ldap', 'AD/LDAP', FALSE, '{}') ON CONFLICT DO NOTHING;

INSERT INTO notification_channel_settings(id, name, password_reset_enabled, approval_enabled, config)
VALUES
  ('email', '邮件', FALSE, FALSE, '{}'),
  ('webhook', 'Webhook', FALSE, FALSE, '{}'),
  ('lark', '飞书机器人', FALSE, FALSE, '{}'),
  ('lark_app', '飞书应用', FALSE, FALSE, '{}'),
  ('wechat', '企业微信机器人', FALSE, FALSE, '{}'),
  ('wechat_app', '企业微信应用', FALSE, FALSE, '{}'),
  ('dingtalk', '钉钉机器人', FALSE, FALSE, '{}'),
  ('dingtalk_app', '钉钉应用', FALSE, FALSE, '{}')
ON CONFLICT (id) DO NOTHING;
