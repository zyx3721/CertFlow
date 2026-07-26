package notify

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"certflow/backend/internal/repository"

	"github.com/robfig/cron/v3"
)

const defaultExpiryNotificationCron = "0 0 * * *"

type ExpiryReminderReport struct {
	Notified int
}

func (s *Service) NotifyExpiryReminders(ctx context.Context) (ExpiryReminderReport, error) {
	report := ExpiryReminderReport{}
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return report, err
	}
	notificationDays := expiryNotificationDays(settings)
	channels, err := s.store.ListNotificationChannelSettings(ctx)
	if err != nil {
		return report, err
	}
	for _, channel := range channels {
		if !channel.ApprovalEnabled {
			continue
		}
		certificates, err := s.store.ListUnnotifiedExpiringCertificates(ctx, time.Now().AddDate(0, 0, notificationDays), channel.ID)
		if err != nil {
			return report, err
		}
		for _, certificate := range certificates {
			if err := s.notifyCertificateExpiry(ctx, settings, certificate, channel); err != nil {
				return report, err
			}
			if err := s.store.MarkCertificateExpiryNotified(ctx, certificate.ID, channel.ID); err != nil {
				return report, err
			}
			report.Notified++
		}
	}
	return report, nil
}

// ExpiryReminderSchedule returns the persisted scan schedule for expiry reminders.
func (s *Service) ExpiryReminderSchedule(ctx context.Context) (string, cron.Schedule, error) {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return "", nil, err
	}
	spec := strings.TrimSpace(stringConfig(settings, "expiryNotificationCron"))
	if spec == "" {
		spec = defaultExpiryNotificationCron
	}
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return "", nil, err
	}
	return spec, schedule, nil
}

func expiryNotificationDays(settings map[string]any) int {
	days := intConfig(settings, "expiryNotificationDays")
	if days < 1 || days > 365 {
		return 15
	}
	return days
}

func (s *Service) notifyCertificateExpiry(ctx context.Context, settings map[string]any, certificate repository.ExpiringCertificate, channel repository.NotificationChannelSetting) error {
	brand := strings.TrimSpace(stringConfig(settings, "siteName"))
	if brand == "" {
		brand = "CertFlow"
	}
	remainingDays := int(math.Ceil(time.Until(certificate.NotAfter).Hours() / 24))
	if remainingDays < 1 {
		remainingDays = 1
	}
	title := brand + " 平台证书到期提醒"
	message := certificateExpiryMessage(certificate.CommonName, certificate.NotAfter, remainingDays)
	if channel.ID == "email" {
		return s.sendExpiryEmail(ctx, channel, title, message, notificationToneNormal)
	}
	return s.sendApprovalChannel(ctx, channel, title, message, notificationToneNormal)
}

func certificateExpiryMessage(commonName string, notAfter time.Time, remainingDays int) string {
	return fmt.Sprintf("证书 %s 将于 %s 到期，剩余 %d 天，请及时续期", commonName, notAfter.Local().Format("2006.01.02 15:04:05"), remainingDays)
}

func (s *Service) NotifyRenewalResult(ctx context.Context, applicant, commonName string, notAfter *time.Time) error {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return err
	}
	brand := strings.TrimSpace(stringConfig(settings, "siteName"))
	if brand == "" {
		brand = "CertFlow"
	}
	title := brand + " 平台证书续期通知"
	message := renewalNotificationMessage(applicant, commonName, notAfter)
	tone := notificationToneNormal
	if notAfter == nil {
		tone = notificationToneWarning
	}
	channels, err := s.store.ListNotificationChannelSettings(ctx)
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if !channel.ApprovalEnabled {
			continue
		}
		if channel.ID == "email" {
			if err := s.sendExpiryEmail(ctx, channel, title, message, tone); err != nil {
				return err
			}
			continue
		}
		if err := s.sendApprovalChannel(ctx, channel, title, message, tone); err != nil {
			return err
		}
	}
	return nil
}

func renewalNotificationMessage(applicant, commonName string, notAfter *time.Time) string {
	prefix := "用户 " + applicant + " 申请的证书 " + commonName
	if notAfter == nil {
		return prefix + " 自动续期失败，请前往平台检查日志"
	}
	return prefix + " 自动续期成功，将于 " + notAfter.Local().Format("2006-01-02 15:04:05") + " 后过期"
}
