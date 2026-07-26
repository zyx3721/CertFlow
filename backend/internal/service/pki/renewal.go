package pki

import (
	"context"
	"time"

	"certflow/backend/internal/domain"
	"certflow/backend/internal/repository"
)

const defaultRenewDays = 10

type AutoRenewalReport struct {
	Enabled              bool
	Renewed              int
	Failed               int
	NotificationFailures int
}

// AutoRenewExpiringCertificates directly issues one replacement certificate for each
// eligible certificate in the configured renewal window.
func (s *Service) AutoRenewExpiringCertificates(ctx context.Context) (AutoRenewalReport, error) {
	report := AutoRenewalReport{}
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return report, err
	}
	if !autoRenewEnabled(settings) {
		return report, nil
	}
	report.Enabled = true
	candidates, err := s.store.ListAutoRenewalCandidates(ctx, time.Now().AddDate(0, 0, renewalDays(settings)))
	if err != nil {
		return report, err
	}
	for _, candidate := range candidates {
		notAfter, err := s.autoRenewCertificate(ctx, candidate)
		if err != nil {
			report.Failed++
			if s.sendRenewalNotification(ctx, candidate.Applicant, candidate.CommonName, nil) {
				report.NotificationFailures++
			}
			continue
		}
		report.Renewed++
		if s.sendRenewalNotification(ctx, candidate.Applicant, candidate.CommonName, &notAfter) {
			report.NotificationFailures++
		}
	}
	return report, nil
}

func (s *Service) autoRenewCertificate(ctx context.Context, candidate repository.AutoRenewalCandidate) (time.Time, error) {
	applicant := domain.User{ID: candidate.ApplicantID, Username: candidate.Applicant}
	request, err := s.createRequest(ctx, RequestInput{
		CAID:         candidate.CAID,
		CommonName:   candidate.CommonName,
		Subject:      candidate.Subject,
		SAN:          candidate.SAN,
		Algorithm:    candidate.Algorithm,
		Purpose:      candidate.Purpose,
		ValidityDays: candidate.ValidityDays,
	}, applicant, "", false)
	if err != nil {
		return time.Time{}, err
	}
	if err := s.store.LinkAutoRenewal(ctx, request.ID, candidate.ID); err != nil {
		_ = s.store.DeleteCertificateRequest(ctx, request.ID)
		return time.Time{}, err
	}
	workflowID, err := s.store.WorkflowIDForCertificate(ctx, request.ID)
	if err != nil {
		_ = s.store.DeleteCertificateRequest(ctx, request.ID)
		return time.Time{}, err
	}
	systemActor := domain.User{}
	if err := s.Approve(ctx, workflowID, "approved", "", systemActor, ""); err != nil {
		_ = s.store.DeleteCertificateRequest(ctx, request.ID)
		return time.Time{}, err
	}
	notAfter, err := s.store.CertificateNotAfter(ctx, request.ID)
	if err != nil {
		return time.Time{}, err
	}
	if err := s.store.DeleteRenewedOriginalCertificate(ctx, candidate.ID); err != nil {
		return time.Time{}, err
	}
	s.store.Audit(ctx, systemActor, "自动续期证书", candidate.CommonName, "certificates", "success", "", autoRenewalAuditDetail(candidate.SerialNumber))
	return notAfter, nil
}

func autoRenewalAuditDetail(serialNumber string) string {
	return "原证书 " + displayCertificateSerial(serialNumber) + " 已自动续期并删除"
}

func (s *Service) sendRenewalNotification(ctx context.Context, applicant, commonName string, notAfter *time.Time) bool {
	notifier, ok := s.notifier.(RenewalNotifier)
	return ok && notifier.NotifyRenewalResult(ctx, applicant, commonName, notAfter) != nil
}

func renewalDays(settings map[string]any) int {
	days, ok := integerSetting(settings["renewDays"])
	if !ok || days < 1 || days > 365 {
		return defaultRenewDays
	}
	return days
}

func autoRenewEnabled(settings map[string]any) bool {
	enabled, _ := settings["autoRenew"].(bool)
	return enabled
}

func integerSetting(value any) (int, bool) {
	switch number := value.(type) {
	case float64:
		return int(number), number == float64(int(number))
	case int:
		return number, true
	default:
		return 0, false
	}
}
