package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"certflow/backend/api/router"
	"certflow/backend/config"
	_ "certflow/backend/docs"
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
	_ = godotenv.Load(filepath.Join(".", ".env"))
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load(logger)
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
		logger.Info("CertFlow backend listening", "addr", cfg.Server.Addr())
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
