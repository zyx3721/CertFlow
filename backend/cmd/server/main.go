package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"certflow/backend/api/router"
	"certflow/backend/config"
	_ "certflow/backend/docs"
	"certflow/backend/internal/buildinfo"
	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
	"certflow/backend/internal/security"
	authsvc "certflow/backend/internal/service/auth"
	notifysvc "certflow/backend/internal/service/notify"
	pkisvc "certflow/backend/internal/service/pki"
	"certflow/backend/pkg/database"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

// flagValues 汇总命令行参数的显式取值，零值表示未设置
type flagValues struct {
	envPath    string
	host       string
	port       string
	mode       string
	dbHost     string
	dbPort     string
	dbName     string
	dbUser     string
	dbPassword string
	dbSSLMode  string
	jwtSecret  string
	sessionTTL int
	pkiKey     string
	corsOrigin string
}

// versionText 组装 -v/-version 输出的版本信息文本
func versionText() string {
	return fmt.Sprintf("certflow %s\ncommit: %s\nbuild: %s\ngo: %s\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate, runtime.Version())
}

// setUsage 自定义 -h/--help 输出：-v 与 -version 合并一行，各参数描述统一换行缩进对齐
func setUsage() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s:\n", os.Args[0])
		flag.VisitAll(func(f *flag.Flag) {
			if f.Name == "version" {
				return
			}
			if f.Name == "v" {
				fmt.Fprintf(w, "  -v, -version\n    \t%s\n", f.Usage)
				return
			}
			name, usage := flag.UnquoteUsage(f)
			fmt.Fprintf(w, "  -%s %s\n    \t%s (default %q)\n", f.Name, name, usage, f.DefValue)
		})
	}
}

// overridesFromFlags 将命令行参数显式值映射为配置加载的覆盖键值，键名与环境变量一致
func overridesFromFlags(v flagValues) map[string]string {
	overrides := make(map[string]string)
	if v.host != "" {
		overrides["SERVER_HOST"] = v.host
	}
	if v.port != "" {
		overrides["SERVER_PORT"] = v.port
	}
	if v.mode != "" {
		overrides["SERVER_MODE"] = v.mode
	}
	if v.dbHost != "" {
		overrides["DB_HOST"] = v.dbHost
	}
	if v.dbPort != "" {
		overrides["DB_PORT"] = v.dbPort
	}
	if v.dbName != "" {
		overrides["DB_NAME"] = v.dbName
	}
	if v.dbUser != "" {
		overrides["DB_USER"] = v.dbUser
	}
	if v.dbPassword != "" {
		overrides["DB_PASSWORD"] = v.dbPassword
	}
	if v.dbSSLMode != "" {
		overrides["DB_SSLMODE"] = v.dbSSLMode
	}
	if v.jwtSecret != "" {
		overrides["JWT_SECRET"] = v.jwtSecret
	}
	if v.sessionTTL > 0 {
		overrides["JWT_EXPIRE_HOURS"] = strconv.Itoa(v.sessionTTL)
	}
	if v.pkiKey != "" {
		overrides["PKI_KEY_ENCRYPTION_KEY"] = v.pkiKey
	}
	if v.corsOrigin != "" {
		overrides["CORS_ORIGIN"] = v.corsOrigin
	}
	return overrides
}

// @title CertFlow API
// @version 1.0
// @description CertFlow 企业内部 PKI 证书生命周期管理平台后端 API，提供认证、CA、证书、CRL、OCSP、审批、审计和系统配置接口。
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	os.Exit(run())
}

func run() int {
	showVersion := flag.Bool("v", false, "显示版本信息并退出")
	flag.BoolVar(showVersion, "version", false, "显示版本信息并退出")
	envPath := flag.String("env", "", ".env 配置文件路径，默认加载工作目录下的 .env")
	host := flag.String("host", "", "后端监听地址，等价环境变量 SERVER_HOST")
	port := flag.String("port", "", "后端监听端口，等价环境变量 SERVER_PORT")
	mode := flag.String("mode", "", "运行模式（release/dev），等价环境变量 SERVER_MODE")
	dbHost := flag.String("db-host", "", "PostgreSQL 主机，等价环境变量 DB_HOST")
	dbPort := flag.String("db-port", "", "PostgreSQL 端口，等价环境变量 DB_PORT")
	dbName := flag.String("db-name", "", "数据库名，等价环境变量 DB_NAME")
	dbUser := flag.String("db-user", "", "数据库用户，等价环境变量 DB_USER")
	dbPassword := flag.String("db-password", "", "数据库密码，等价环境变量 DB_PASSWORD")
	dbSSLMode := flag.String("db-sslmode", "", "数据库 SSL 模式，等价环境变量 DB_SSLMODE")
	jwtSecret := flag.String("jwt-secret", "", "会话签名密钥，等价环境变量 JWT_SECRET")
	sessionTTL := flag.Int("session-ttl", 0, "会话有效期（小时），等价环境变量 JWT_EXPIRE_HOURS")
	pkiKey := flag.String("pki-key", "", "私钥加密主密钥（Base64 编码 32 字节），等价环境变量 PKI_KEY_ENCRYPTION_KEY")
	corsOrigin := flag.String("cors-origin", "", "允许的跨域来源，等价环境变量 CORS_ORIGIN")
	setUsage()
	flag.Parse()

	if *showVersion {
		fmt.Print(versionText())
		return 0
	}

	flags := flagValues{
		envPath:    *envPath,
		host:       *host,
		port:       *port,
		mode:       *mode,
		dbHost:     *dbHost,
		dbPort:     *dbPort,
		dbName:     *dbName,
		dbUser:     *dbUser,
		dbPassword: *dbPassword,
		dbSSLMode:  *dbSSLMode,
		jwtSecret:  *jwtSecret,
		sessionTTL: *sessionTTL,
		pkiKey:     *pkiKey,
		corsOrigin: *corsOrigin,
	}
	if flags.envPath != "" {
		_ = godotenv.Load(flags.envPath)
	} else {
		_ = godotenv.Load(filepath.Join(".", ".env"))
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.LoadWithOverrides(overridesFromFlags(flags), logger)
	if err != nil {
		logger.Error("Load configuration failed", "error", err)
		return 1
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Error("Connect database failed", "error", err)
		return 1
	}
	defer pool.Close()
	if err = database.Migrate(ctx, pool); err != nil {
		logger.Error("Migrate database failed", "error", err)
		return 1
	}
	store := repository.New(pool)
	if err = store.EnsureDefaultAdmin(ctx); err != nil {
		logger.Error("Initialize default admin failed", "error", err)
		return 1
	}
	box, err := security.New(cfg.PKI.KeyEncryptionKey)
	if err != nil {
		logger.Error("Initialize key encryption failed", "error", err)
		return 1
	}
	notifyService := notifysvc.New(store, box)
	pkiService := pkisvc.New(store, box, notifyService)
	if err = pkiService.EnsureOCSPResponders(ctx); err != nil {
		logger.Error("Initialize OCSP responders failed", "error", err)
		return 1
	}
	notificationContext, stopNotifications := context.WithCancel(context.Background())
	defer stopNotifications()
	go runExpiryReminderLoop(notificationContext, store, pkiService, notifyService, logger)
	handler := router.New(
		cfg,
		store,
		authsvc.New(store, cfg.Auth.SessionTTL(), cfg.Auth.SessionSecret, box, notifyService),
		pkiService,
		notifyService,
		logger,
	)
	server := &http.Server{Addr: cfg.Server.Addr(), Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("CertFlow backend listening", "addr", cfg.Server.Addr(), "version", buildinfo.Version)
		serverErrors <- server.ListenAndServe()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case <-signals:
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server stopped unexpectedly", "error", err)
			return 1
		}
		return 0
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stopNotifications()
	if err := server.Shutdown(shutdown); err != nil {
		logger.Error("Shutdown server failed", "error", err)
		return 1
	}
	return 0
}

func runExpiryReminderLoop(ctx context.Context, store *repository.Store, pki *pkisvc.Service, service *notifysvc.Service, logger *slog.Logger) {
	var lastSpec string
	var lastRun time.Time
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if runExpiryReminderIfDue(ctx, store, pki, service, logger, time.Now(), &lastSpec, &lastRun) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func runExpiryReminderIfDue(ctx context.Context, store *repository.Store, pki *pkisvc.Service, service *notifysvc.Service, logger *slog.Logger, now time.Time, lastSpec *string, lastRun *time.Time) bool {
	spec, schedule, err := service.ExpiryReminderSchedule(ctx)
	if err != nil {
		logger.Error("Load certificate expiry schedule failed", "error", err)
		return false
	}
	if spec != *lastSpec {
		*lastSpec = spec
		*lastRun = time.Time{}
	}
	runAt := expiryReminderRunAt(schedule, now)
	if runAt.After(now) || runAt.Equal(*lastRun) {
		return false
	}
	expiryReport, expiryErr := service.NotifyExpiryReminders(ctx)
	if expiryErr != nil {
		logger.Error("Send certificate expiry reminders failed", "error", expiryErr)
	}
	renewalReport, renewalErr := pki.AutoRenewExpiringCertificates(ctx)
	if renewalErr != nil {
		logger.Error("Automatically renew certificates failed", "error", renewalErr)
	}
	if renewalReport.NotificationFailures > 0 {
		logger.Error("Send certificate renewal notification failed", "count", renewalReport.NotificationFailures)
	}
	result := "success"
	if renewalErr != nil || expiryErr != nil || renewalReport.Failed > 0 || renewalReport.NotificationFailures > 0 {
		result = "failure"
	}
	store.Audit(ctx, domain.User{}, "执行到期扫描", "证书续期与提醒", "certificates", result, "", expiryScanAuditDetail(spec, expiryReport.Notified, renewalReport, renewalErr, expiryErr))
	*lastRun = runAt
	return true
}

func expiryReminderRunAt(schedule cron.Schedule, now time.Time) time.Time {
	return schedule.Next(now.Truncate(time.Minute).Add(-time.Minute))
}

func expiryScanAuditDetail(spec string, notified int, renewal pkisvc.AutoRenewalReport, renewalErr, expiryErr error) string {
	detail := fmt.Sprintf("扫描计划 %s 已执行，到期提醒已发送 %d 次", spec, notified)
	if renewalErr != nil {
		detail += "，自动续期执行异常"
	} else if !renewal.Enabled {
		detail += "，自动续期未启用"
	} else {
		detail += fmt.Sprintf("，自动续期成功 %d 张，失败 %d 张", renewal.Renewed, renewal.Failed)
	}
	if renewal.NotificationFailures > 0 {
		detail += fmt.Sprintf("，续期站外通知失败 %d 次", renewal.NotificationFailures)
	}
	if expiryErr != nil {
		detail += "，到期提醒发送异常"
	}
	return detail
}
