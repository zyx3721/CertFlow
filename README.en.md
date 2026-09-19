<div align="center">

<h1>CertFlow</h1>

<p><b>Self-hosted PKI certificate lifecycle management</b> — CA hierarchy · issuance · approval · revocation · CRL · OCSP · audit</p>

<p><a href="README.md">简体中文</a> · <b>English</b></p>

Internal HTTPS, client and mTLS certificates usually live on ops USB sticks, spreadsheets and verbal approvals.  
CertFlow turns them into a **self-hosted** trust fabric: a Go backend, a React console and PostgreSQL.  
CA hierarchies, certificate request & approval, revocation and CRL/OCSP publishing in one place, with private keys always encrypted at rest.  
No tokens, no passwords, no real hostnames committed to the repository.

<p>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?labelColor=1f2937" alt="MIT License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white&labelColor=1f2937" alt="Go 1.25+"></a>
  <a href="https://react.dev/"><img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white&labelColor=1f2937" alt="React 19"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16+-4169E1?logo=postgresql&logoColor=white&labelColor=1f2937" alt="PostgreSQL"></a>
  <img src="https://img.shields.io/badge/permissions-27-059669?labelColor=1f2937" alt="27 permissions">
</p>

<p>
  <b><a href="#preview">Preview</a></b> ·
  <a href="#what-it-does">What it does</a> ·
  <a href="#how-it-works">How it works</a> ·
  <a href="#tech-stack">Tech stack</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#deployment">Deployment</a> ·
  <a href="#permission-model">Permissions</a> ·
  <a href="#security">Security</a> ·
  <a href="#faq">FAQ</a> ·
  <a href="#repository-layout">Layout</a> ·
  <a href="#docs">Docs</a>
</p>

</div>

---

## Preview

### Login

An internal system with no public sign-up. Supports local passwords, AD/LDAP and WeCom (Enterprise WeChat) QR login (direct or unified auth-center mode); password reset uses a captcha + email code.

![Login](.github/images/certflow-login.jpg)

### Dashboard

Read-only summary: CA and certificate totals, issuance/revocation trends and recent audit activity. The sidebar shows entries based on the signed-in user's permissions; anything the user cannot do is hidden.

![Home](.github/images/certflow-home.jpg)

## What it does

- **CA management** — Create or import Root, Intermediate and Issuing CAs with RSA, ECDSA or ED25519 keys; visual CA tree, PEM certificate viewing and download; a CA branch can only be deleted recursively when no certificate references it, keeping every issuing chain traceable.
- **Certificate lifecycle** — Requests with Subject, SAN, usage (server/client/mTLS), validity and optional CSR; every submission creates an approval record and approval triggers automatic serial generation and issuance; material download returns CRT, bundle ZIP or the rejected request's CSR depending on status; supports verification, revocation with standard reasons, and deletion.
- **CRL & OCSP** — On-demand signed X.509 CRL downloads (single `.crl` or multi-CA ZIP); RFC 6960 binary OCSP responses signed by a dedicated responder certificate per issuing CA, plus JSON status queries; revocation history is kept for continuous CRL/OCSP verification.
- **Approvals & audit** — Certificate requests automatically create approval records; login/logout, CA, certificate, approval, user and system configuration changes are fully audited with search, filtering and export.
- **Expiry reminders & auto-renewal** — Cron-driven scans push reminders to email, Webhook, Lark, WeCom Work and DingTalk channels; with auto-renewal enabled, renewal certificates with fresh key material are issued inside the renewal window.
- **Users & permissions** — Local password, AD/LDAP, WeCom QR login (direct or unified auth-center SSO), WeCom account binding, and an email-based password reset flow; built-in `admin` / `operator` / `viewer` roles, custom roles selecting from 27 permissions, and user-group based batch authorization. The frontend hides unauthorized entries while the backend checks every endpoint.
- **System settings** — Branding, CRL/OCSP service parameters, renewal and expiry reminders, password-reset TTLs, WeCom QR validity, users/groups/roles, AD/LDAP and WeCom authentication, notification channels, with LDAP connectivity and email delivery tests.
- **API docs** — Integrated swag + Swagger UI covering endpoints, parameters, auth and responses out of the box.

**It is not** a certificate transparency log or a scanner: CertFlow manages "internal CAs + certificate inventory + approvals + revocation proof". It does not scan the network for certificates, does not implement ACME auto-issuance, and does not proxy external cloud KMS.

The same internal certificate ledger, two ways:

| Scenario | Manual management | With CertFlow |
| --- | --- | --- |
| Issue a service certificate | Ops runs openssl by hand | Fill in Subject/SAN, submit for approval |
| Who approved it | Dig through chat history | Approval records and full audit trail |
| Expiry | Rely on memory | Automatic reminders, optional auto-renewal |
| Handing out private keys | ZIP files everywhere | Keys encrypted with AES-256-GCM, authorized download endpoints only |
| Proving a revocation | Emails to every team | Standard CRL/OCSP endpoints for consumers |
| CA private key | On someone's laptop | Encrypted at rest, never echoed by APIs |
| Permissions | All or nothing | 27 permissions checked per endpoint |

## How it works

```text
        Browser
           │  http
           ▼
  ┌──────────────────────────────────────┐
  │  Nginx (in-container or host)        │
  │  /assets/ · /       → frontend SSR   │
  │  /api/ · /swagger/  → Go backend     │
  │  /crl/ · /ocsp      → Go backend     │
  └──────────────────────────────────────┘
        │                        │
        ▼                        ▼
  React 19 console          Go 1.25 backend
  Nitro SSR :5173       net/http :8080
                                 │
                          PostgreSQL 16+
                users · CAs · certificates · audit
```

- **Who does what** — The console reads the database to render CAs, certificates and statistics; changes such as CA creation, certificate requests, approvals, revocations and system settings are validated by the backend and written to the audit log.
- **Where data lives** — PostgreSQL is the source of truth; embedded migrations run in filename order on startup; certificate private keys are encrypted with the application master key before being stored.
- **Secrets** — Session tokens are stored as SHA-256 hashes only; private keys, LDAP bind passwords, SMTP passwords and WeCom secrets are encrypted at rest and never echoed, APIs only expose whether they are configured.
- **Public surface** — Only health checks, branding, login, WeCom authorization callbacks, password reset and public CRL/OCSP endpoints are anonymous; everything else requires `Authorization: Bearer <token>`.

## Tech stack

| Layer | Choice |
| --- | --- |
| Backend language | Go 1.25+ |
| HTTP routing | Standard library `net/http` and `ServeMux` |
| Database | PostgreSQL 16+ ([pgx/v5](https://github.com/jackc/pgx) pool) |
| Auth & crypto | Opaque session tokens + bcrypt, AD/LDAP (go-ldap/ldap v3), WeCom OAuth QR login (direct / unified auth-center SSO), AES-256-GCM secret encryption |
| PKI | Standard library `crypto/x509`, `golang.org/x/crypto/ocsp`, on-the-fly CRL signing |
| API docs | swag + http-swagger (Swagger UI) |
| Frontend | React 19 + TanStack Start / Router / Query |
| Language & build | TypeScript 5 + Vite + Nitro |
| Styling & UI | Tailwind CSS, Radix UI, lucide-react, sonner |
| Runtime packaging | Docker (Nginx + Supervisor multi-process) |

## Quick start

Local development needs **Go 1.25+**, **Node.js 20+** and **PostgreSQL 16+**.

```bash
git clone https://github.com/zyx3721/CertFlow.git
cd CertFlow
```

**Prepare the database**

```bash
psql -Upostgres -c "CREATE DATABASE certflow;"
```

No PostgreSQL around? Start one with Docker:

```bash
docker run -d --name pg-prod \
  -p 5432:5432 \
  -v /data/PgSqlData:/var/lib/postgresql/data \
  -e POSTGRES_PASSWORD="123456ok!" \
  -e LANG=C.UTF-8 -e TZ=Asia/Shanghai \
  postgres:17-alpine
```

**Backend**

```bash
cd backend
go mod download
cp .env.example .env      # configure the database; generate keys with: openssl rand -base64 32
go run cmd/server/main.go
```

The backend listens on `http://localhost:8080` by default. On first start it runs the migrations and creates the default administrator `admin / 123456`.

**Frontend** (another terminal)

```bash
cd frontend
npm install
npm run dev
```

The frontend runs on `http://localhost:5173` (if the backend port is not 8080, set `VITE_API_BASE_URL` in `frontend/.env`).

Open `http://localhost:5173`, sign in with `admin / 123456`, and **change the password immediately**. Swagger is available at `http://localhost:8080/swagger/index.html`.

## Deployment

Only two paths are documented: **Docker Compose** (recommended) and **Release binaries**. The full step-by-step procedures (host Nginx HTTP/HTTPS examples) live in [docs/manual.md](docs/manual.md).

### Option 1: Docker Compose

The image bundles the Go backend, Nginx and the frontend Nitro SSR under Supervisor, exposing only port 80:

```bash
git clone https://github.com/zyx3721/CertFlow.git
cd CertFlow/deploy
cp .env.example .env && vim .env    # configure database, JWT_SECRET and PKI_KEY_ENCRYPTION_KEY
docker compose up -d
```

Key `.env` entries (full list in [docs/manual.md](docs/manual.md), section 5.5):

| Variable | Purpose |
| --- | --- |
| `JWT_SECRET` | Session signing secret; set a long random value in production |
| `PKI_KEY_ENCRYPTION_KEY` | Master key encrypting private keys and secrets, Base64-encoded 32 bytes; generate with `openssl rand -base64 32`. **If lost, encrypted private keys cannot be recovered** |
| `DB_HOST` / `DB_PASSWORD` etc. | PostgreSQL connection; a `postgres` container ships with the compose file — comment it out to use an external database |

Service management:

```bash
docker compose ps                    # status
docker compose logs -f certflow      # logs
docker compose restart certflow      # restart
docker compose down                  # stop
```

**Access**

- Console: `http://your-host/`, default account `admin / 123456`
- API docs: `http://your-host/swagger/index.html`
- Health check: `http://your-host/health`

To unify domains, HTTPS or multi-site routing on the host, map a non-80 port (`8080:80`) and reverse-proxy from the host Nginx. Make sure the public PKI paths (`/crl/`, `/ocsp`) are proxied to the backend as well.

### Option 2: Release binaries

Head to the [GitHub Releases](https://github.com/zyx3721/CertFlow/releases) page, download the archive matching your operating system and CPU architecture, then verify, extract, configure and start as described below.

**Which package to download**

| Your machine | File |
| --- | --- |
| Linux x86_64 | `certflow_<version>_linux_amd64.tar.gz` |
| Linux ARM64 (Kunpeng, Phytium, …) | `certflow_<version>_linux_arm64.tar.gz` |
| macOS Intel | `certflow_<version>_darwin_amd64.tar.gz` |
| macOS Apple silicon | `certflow_<version>_darwin_arm64.tar.gz` |
| Windows x86_64 | `certflow_<version>_windows_amd64.zip` |
| Windows ARM64 | `certflow_<version>_windows_arm64.zip` |
| Frontend UI (required on every platform) | `certflow-frontend_<version>.tar.gz` |
| Checksums | `SHA256SUMS` |

The backend package contains the `certflow` executable (`certflow.exe` on Windows), `.env.example` and a `README.txt`; the frontend package contains the Nitro SSR `.output` artifacts. The backend binary has no runtime dependencies; the frontend SSR requires Node.js on the target machine, and both options require you to provide PostgreSQL 16+.

**1. Verify the download**

```bash
VERSION=1.0.3
mkdir -p /data/certflow && cd /data/certflow
sha256sum -c SHA256SUMS
```

**2. Extract**

```bash
mkdir -p backend frontend/.output
tar -xzf certflow_${VERSION}_linux_amd64.tar.gz -C backend --strip-components=1
tar -xzf certflow-frontend_${VERSION}.tar.gz -C frontend/.output
```

This yields the following layout:

```text
/data/certflow/
├── backend/
│   ├── certflow           # backend binary
│   └── .env.example
└── frontend/
    └── .output/
        ├── public/        # browser static assets
        └── server/
            └── index.mjs  # Nitro SSR entry
```

**3. Configure and start the backend**

```bash
cd /data/certflow/backend
cp .env.example .env
vim .env               # at minimum set JWT_SECRET and PKI_KEY_ENCRYPTION_KEY, and point to a reachable PostgreSQL
./certflow
```

Use systemd for long-running deployments:

```ini
# /etc/systemd/system/certflow-backend.service
[Unit]
Description=CertFlow Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=/data/certflow/backend
ExecStart=/data/certflow/backend/certflow
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload && systemctl enable --now certflow-backend
```

**4. Start the frontend SSR**

```bash
cd /data/certflow/frontend
HOST=127.0.0.1 PORT=5173 node .output/server/index.mjs
```

**5. Terminate with Nginx**

```nginx
server {
    listen 80;
    server_name your-domain.com;
    client_max_body_size 50m;

    # Frontend static assets served straight from .output/public
    location ^~ /assets/ {
        root /data/certflow/frontend/.output/public;
        try_files $uri =404;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /swagger/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
    }

    # Public PKI distribution: CRL and OCSP must keep their original paths and methods
    location ^~ /crl/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        add_header X-Content-Type-Options nosniff always;
    }

    location = /ocsp {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        add_header X-Content-Type-Options nosniff always;
    }

    location ^~ /ocsp/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        add_header X-Content-Type-Options nosniff always;
    }

    location / {
        proxy_pass http://127.0.0.1:5173;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location = /health {
        proxy_pass http://127.0.0.1:8080/api/health;
    }
}
```

The frontend must be served by `node .output/server/index.mjs`; **pointing Nginx at `.output/public` only will break server-rendered pages**. Full examples with HTTPS and 80→443 redirection are in [docs/manual.md](docs/manual.md), section 4.4.

**6. Access**

Same as Docker: console `http://your-domain.com` (`admin / 123456`), API docs `/swagger/index.html`, health check `/health`.

## Permission model

Every endpoint checks role permissions and returns 403 when missing; the `admin` user has all permissions. Permission keys look like `module.resource.action` — 27 in total.

| Identity | Default permissions |
| --- | --- |
| Default administrator `admin` | All 27; the default account cannot be deleted |
| Built-in role `admin` | All 27 |
| Built-in role `operator` | Full operations on dashboard, CAs, certificates and approvals, audit viewing and export; no system settings permissions |
| Built-in role `viewer` | Read-only: dashboard, CAs, certificates, CRL and audit |
| Custom roles / user groups | Pick from the 27; manage permissions imply their read counterpart; user groups grant in batch |

By module:

- **Dashboard** — `dashboard.read`
- **CA management** — `ca.{read|download|add|delete}`
- **Certificates** — `certificates.{read|request|download|verify|manage|revoke|delete}`; requesting implies certificate and CA read
- **Approvals** — `workflows.{read|approve|delete}`
- **Revocation** — `crl.{read|manage}`
- **Audit** — `audit.{read|manage}` (export implies view)
- **System settings** — `settings.{base|users|auth|notifications}.read` / `.manage`

## Security

```text
Console accounts sign in, change settings, manage certificates
        +
Bearer token checked against 27 permissions per endpoint, 403 otherwise
        +
Private keys and secrets encrypted with AES-256-GCM, responses redacted
        +
No tokens / passwords / real hostnames committed to the repository
```

- **Change the default password first** — Update the `admin` password immediately after the first deployment.
- **Set both keys explicitly** — Without `JWT_SECRET` the backend only generates a per-process temporary key that invalidates all sessions on restart; a wrong or missing `PKI_KEY_ENCRYPTION_KEY` prevents startup, and keys encrypted with it cannot be recovered — back it up.
- **Enable HTTPS** — Terminate TLS on Nginx in production; see [docs/manual.md](docs/manual.md) section 4.4.2.
- **Tighten CORS** — Configure `CORS_ORIGIN` per environment; do not keep `*`.
- **Private key boundary** — Private keys never appear in CA or certificate list APIs; downloads go through dedicated authorized endpoints and are audited.
- **Authentication security** — LDAP bind passwords, SMTP passwords and WeCom secrets (direct app secret, auth-center app secret) share the master-key encryption and are never echoed; WeCom OAuth states are HMAC-SHA256 signed with the session secret (default 5 minutes, adjustable 1-60); auth-center tickets are single-use and verified via HMAC signature plus a timestamp window.
- **Traceability** — Logins (including WeCom scans with failure reasons), bindings/unbindings, CA, certificate, approval, user and settings changes are all recorded with time, user, module, target, source IP, result and non-sensitive details.

## API docs

The backend ships with Swagger/OpenAPI:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **OpenAPI JSON**: `http://localhost:8080/swagger/doc.json`
- **Health check**: `GET /api/health`

Anonymous endpoints are limited to: `GET /api/v1/auth/providers`, `GET /api/v1/auth/wecom/authorize`, `POST /api/v1/auth/wecom/callback`, `POST /api/v1/auth/wecom/sso/callback`, `GET /api/v1/auth/password-reset/captcha`, `POST /api/v1/auth/password-reset/verify`, `POST /api/v1/auth/password-reset/send`, `POST /api/v1/auth/password-reset/confirm`, `GET /api/v1/public/settings`, `GET /api/health`, plus the public PKI endpoints `GET /crl/{caId}.crl`, `POST /ocsp`, `GET /ocsp`, `GET /ocsp/{serial}`, `GET /ocsp/health`; everything else requires `Authorization: Bearer <token>`.

Login request example:

```json
{
  "username": "admin",
  "password": "123456",
  "provider": "local"
}
```

The full endpoint list grouped by module (health, authentication, password reset, CAs, certificates, CRL, approvals, audit, system settings, public PKI) is available in [docs/manual.md](docs/manual.md), chapter "API docs".

After changing an endpoint, regenerate the Swagger artifacts inside `backend/`:

```bash
swag init -g cmd/server/main.go -o docs
```

## Database

A PostgreSQL database with 20 tables, created by embedded migrations in filename order (`backend/pkg/database/migrations/`); applied migrations are tracked in `schema_migrations` and never replayed.

| Group | Tables |
| --- | --- |
| Users & sessions | `users`, `sessions`, `roles`, `user_roles`, `user_groups`, `user_group_members`, `user_group_roles`, `user_wecom_bindings` |
| PKI business | `certificate_authorities`, `certificates`, `certificate_revocation_entries`, `ocsp_responders`, `workflows` |
| System & settings | `system_settings`, `auth_provider_settings`, `notification_channel_settings`, `certificate_expiry_notifications` |
| Audit & password reset | `audit_entries`, `password_reset_requests` |

Column-level details are described in [docs/manual.md](docs/manual.md) alongside the `001_init.sql` schema.

## FAQ

**Forgot the administrator password?**

Update the database directly (recommended, no data loss). First generate a bcrypt hash for the new password (`123456` produces `$2a$10$y6sEolCX.y.We871sMtwkO2MkT4dUUJhafNtMZeLwbT07DV62JkuS`), then:

```bash
psql -Upostgres -d certflow -c 'UPDATE users SET password_hash = ''$2a$10$y6sEolCX.y.We871sMtwkO2MkT4dUUJhafNtMZeLwbT07DV62JkuS'' WHERE username = ''admin'';'
```

**How long do sessions last?**

24 hours by default (`JWT_EXPIRE_HOURS`) with an additional 12-hour idle timeout (`SESSION_IDLE_TIMEOUT_HOURS`); restart the backend after changing them. Tokens are stored hashed, so rotating `JWT_SECRET` does not sign anyone out — truncate the `sessions` table to force a global sign-out.

**How do I enable LDAP login?**

First fill in the connection parameters under "System settings → Authentication → AD/LDAP" and enable it (the "Test" button verifies connectivity and matched users); then create users in "User settings" whose usernames match the LDAP `sAMAccountName`. When no such local user exists, login is rejected even with valid LDAP credentials.

**How do I enable WeCom QR login?**

Create a self-built app in the WeCom admin console and note the AgentID and Secret; add this system's domain to the app's trusted web authorization domain and add the server egress IP to the trusted IP list. Then, under "System settings → Authentication → WeCom", choose the "Direct" mode, fill in the corp ID, AgentID and Secret, and enable (the redirect prefix can be left empty to infer from the current access address). Users sign in with their password once, use "Bind WeCom" in the top-right menu to scan the QR code, and can then choose WeCom QR login on the login page.

If your organization already runs a unified auth center (wecom-auth-center), switch the "Auth mode" to "Unified auth center" and fill in the center's address, app ID and app secret; on the center side, register this system's `domain` (external address) and `callback_path` (`/login`). Multiple internal systems can share one WeCom app configuration while binding and login flows stay identical.

**Can the database be moved directly?**

Yes. Stop the service and use `pg_dump` / `pg_restore` (or `CREATE DATABASE ... TEMPLATE`); on the application side keep `JWT_SECRET` and `PKI_KEY_ENCRYPTION_KEY` identical, otherwise encrypted private keys cannot be decrypted.

**Why can't I serve the built frontend as a static site?**

The console is a TanStack Start + Nitro SSR application — pages are rendered by `node .output/server/index.mjs`; the static directory only serves `/assets/` and similar resources.

For everything else, see [docs/manual.md](docs/manual.md) and the [beginner's guide](docs/CertFlow新手证书签发与Nginx配置指南.md).

## Repository layout

```text
CertFlow/
├── backend/                 Go API, PKI services and PostgreSQL migrations
│   ├── api/router/          HTTP boundary, auth and permission checks, Swagger annotations and models
│   ├── config/              Environment variable loading and validation
│   ├── docs/                Generated Swagger/OpenAPI files
│   ├── internal/            Domain models, repositories, security and services
│   ├── pkg/database/        PostgreSQL connection, initialization and migrations
│   └── cmd/server/          Service entry point
├── deploy/                  Docker Compose, Nginx and process management files
├── docs/                    Full documentation and the beginner's guide
├── frontend/                TanStack Start React console
│   └── src/
│       ├── components/      App layout, notification center, dialogs and base UI
│       ├── features/        Auth, PKI, settings pages and role editor components
│       ├── lib/             Auth, PKI and settings API clients and utilities
│       ├── routes/          TanStack Router route stubs
│       ├── router.tsx       Router instance
│       ├── start.ts         React Start client entry
│       └── server.ts        Server entry and error handling
├── .github/                 GitHub Actions workflows and preview images
├── .dockerignore            Docker build ignore rules
├── .gitignore               Git ignore rules
├── LICENSE
├── README.md                Chinese readme (this repo's main readme)
├── README.en.md             English
└── docs/manual.md           Full documentation (complete endpoint list, Nginx and HTTPS examples)
```

## Docs

| Start here | Then |
| --- | --- |
| [Quick start](#quick-start) | Run the backend and frontend locally, default account and ports |
| [Deployment](#deployment) | Docker Compose and Release binaries, environment variables, reverse proxy |
| [Permission model](#permission-model) | How the 27 permissions are grouped, what built-in roles get |
| [Full documentation](docs/manual.md) | Complete endpoint list, full Nginx and HTTPS examples, environment variables |
| [Beginner's guide](docs/CertFlow新手证书签发与Nginx配置指南.md) | From creating an internal CA to issuing certificates behind Nginx HTTPS |
| [中文 README](README.md) | The same content in Chinese |

## Version history

| Version | Date | Changelog |
| --- | --- | --- |
| v1.1.0 | 2026-09-20 | [verchanglog/v1.1.0.md](verchanglog/v1.1.0.md) |
| v1.0.3 | 2026-07-28 | [verchanglog/v1.0.3.md](verchanglog/v1.0.3.md) |
| v1.0.2 | 2026-07-26 | [verchanglog/v1.0.2.md](verchanglog/v1.0.2.md) |
| v1.0.1 | 2026-07-26 | [verchanglog/v1.0.1.md](verchanglog/v1.0.1.md) |
| v1.0.0 | 2026-07-26 | [verchanglog/v1.0.0.md](verchanglog/v1.0.0.md) |

Build artifacts and release notes for each version are on [GitHub Releases](https://github.com/zyx3721/CertFlow/releases).

## Acknowledgements

Thanks to the following open-source projects and communities:

- [jackc/pgx](https://github.com/jackc/pgx) — High-performance PostgreSQL driver and pool
- [swaggo/swag](https://github.com/swaggo/swag) — Swagger documentation generator
- [go-ldap/ldap](https://github.com/go-ldap/ldap) — LDAP v3 client
- [TanStack](https://tanstack.com/) — Router / Query / Start frontend suite
- [Tailwind CSS](https://tailwindcss.com/) — Utility-first CSS framework

## License

This project is released under the [MIT License](LICENSE): free to use, copy, modify, merge, publish, distribute, sublicense and sell, provided the copyright and permission notices are retained in all copies or substantial parts.

## Contact

- **Email**: 416685476@qq.com
- **GitHub Issues**: [zyx3721/CertFlow/issues](https://github.com/zyx3721/CertFlow/issues)
- **Project home**: [github.com/zyx3721/CertFlow](https://github.com/zyx3721/CertFlow)

---

**⭐ If this project helps you, a Star is much appreciated!**
