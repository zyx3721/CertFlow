package domain

import "time"

type User struct {
	ID                     string     `json:"id"`
	Username               string     `json:"username"`
	DisplayName            string     `json:"displayName"`
	Email                  string     `json:"email"`
	Role                   string     `json:"role"`
	Source                 string     `json:"source"`
	AuthenticationProvider string     `json:"-"`
	Roles                  []Role     `json:"roles"`
	DirectRoles            []Role     `json:"directRoles"`
	Disabled               bool       `json:"disabled"`
	WecomBound             bool       `json:"wecomBound"`
	LastLoginAt            *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
	Permissions            []string   `json:"permissions"`
}

type Role struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	Builtin     bool      `json:"builtin"`
	Disabled    bool      `json:"disabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UserGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Disabled    bool      `json:"disabled"`
	Members     []User    `json:"members"`
	Roles       []Role    `json:"roles"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Permission struct {
	Key                   string   `json:"key"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Category              string   `json:"category"`
	ImpliedReadPermission string   `json:"impliedReadPermission,omitempty"`
	ImpliedPermissions    []string `json:"impliedPermissions,omitempty"`
}

type PasswordResetCaptcha struct {
	Token     string    `json:"token"`
	Question  string    `json:"question"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type PasswordResetChannel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MaskedTo   string `json:"maskedTo"`
	RequiresTo bool   `json:"requiresTo"`
}

type Session struct {
	Token     string    `json:"token"`
	User      User      `json:"user"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type CA struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Algorithm      string    `json:"algorithm"`
	Subject        string    `json:"subject"`
	ParentID       string    `json:"parentId"`
	CertificatePEM string    `json:"certificatePem,omitempty"`
	PrivateKeyPEM  string    `json:"-"`
	NotBefore      time.Time `json:"notBefore"`
	NotAfter       time.Time `json:"notAfter"`
	IssuedCerts    int       `json:"issuedCerts"`
	RevokedCerts   int       `json:"revokedCerts"`
	Status         string    `json:"status"`
}

type Certificate struct {
	ID             string     `json:"id"`
	CAID           string     `json:"caId"`
	SerialNumber   string     `json:"serialNumber"`
	CommonName     string     `json:"commonName"`
	Subject        string     `json:"subject"`
	Algorithm      string     `json:"algorithm"`
	Purpose        string     `json:"purpose"`
	Status         string     `json:"status"`
	Applicant      string     `json:"applicant"`
	Source         string     `json:"source"`
	Fingerprint    string     `json:"fingerprint,omitempty"`
	CertificatePEM string     `json:"certificatePem,omitempty"`
	PrivateKeyPEM  string     `json:"privateKeyPem,omitempty"`
	CSRPEM         string     `json:"csrPem,omitempty"`
	RejectReason   string     `json:"rejectReason,omitempty"`
	SAN            []string   `json:"san"`
	NotBefore      time.Time  `json:"notBefore"`
	NotAfter       time.Time  `json:"notAfter"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	RevokeReason   string     `json:"revokeReason,omitempty"`
}

type Workflow struct {
	ID            string    `json:"id"`
	CertificateID string    `json:"certificateId"`
	Applicant     string    `json:"applicant"`
	CertName      string    `json:"certName"`
	Status        string    `json:"status"`
	RejectReason  string    `json:"rejectReason,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Audit struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Module    string    `json:"module"`
	Result    string    `json:"result"`
	IP        string    `json:"ip"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
}

type CRLEntry struct {
	ID           string    `json:"id"`
	SerialNumber string    `json:"serialNumber"`
	RevokedAt    time.Time `json:"revokedAt"`
	Reason       string    `json:"reason"`
	CAID         string    `json:"caId"`
	CAName       string    `json:"caName"`
	CommonName   string    `json:"commonName"`
}

const (
	PermissionDashboardRead        = "dashboard.read"
	PermissionCARead               = "ca.read"
	PermissionCADownload           = "ca.download"
	PermissionCAAdd                = "ca.add"
	PermissionCADelete             = "ca.delete"
	PermissionCertificateRead      = "certificates.read"
	PermissionCertificateRequest   = "certificates.request"
	PermissionCertificateDownload  = "certificates.download"
	PermissionCertificateVerify    = "certificates.verify"
	PermissionCertificateManage    = "certificates.manage"
	PermissionCertificateRevoke    = "certificates.revoke"
	PermissionCertificateDelete    = "certificates.delete"
	PermissionWorkflowRead         = "workflows.read"
	PermissionWorkflowApprove      = "workflows.approve"
	PermissionWorkflowDelete       = "workflows.delete"
	PermissionCRLRead              = "crl.read"
	PermissionCRLManage            = "crl.manage"
	PermissionAuditRead            = "audit.read"
	PermissionAuditManage          = "audit.manage"
	PermissionSettingsBaseRead     = "settings.base.read"
	PermissionSettingsBaseManage   = "settings.base.manage"
	PermissionSettingsUsersRead    = "settings.users.read"
	PermissionSettingsUsersManage  = "settings.users.manage"
	PermissionSettingsAuthRead     = "settings.auth.read"
	PermissionSettingsAuthManage   = "settings.auth.manage"
	PermissionSettingsNotifyRead   = "settings.notifications.read"
	PermissionSettingsNotifyManage = "settings.notifications.manage"
)

func PermissionsForRole(role string) []string {
	switch role {
	case "admin":
		return []string{
			PermissionDashboardRead,
			PermissionCARead,
			PermissionCADownload,
			PermissionCAAdd,
			PermissionCADelete,
			PermissionCertificateRead,
			PermissionCertificateRequest,
			PermissionCertificateDownload,
			PermissionCertificateVerify,
			PermissionCertificateManage,
			PermissionCertificateRevoke,
			PermissionCertificateDelete,
			PermissionWorkflowRead,
			PermissionWorkflowApprove,
			PermissionWorkflowDelete,
			PermissionCRLRead,
			PermissionCRLManage,
			PermissionAuditRead,
			PermissionAuditManage,
			PermissionSettingsBaseRead,
			PermissionSettingsBaseManage,
			PermissionSettingsUsersRead,
			PermissionSettingsUsersManage,
			PermissionSettingsAuthRead,
			PermissionSettingsAuthManage,
			PermissionSettingsNotifyRead,
			PermissionSettingsNotifyManage,
		}
	case "operator":
		return []string{
			PermissionDashboardRead,
			PermissionCARead,
			PermissionCADownload,
			PermissionCAAdd,
			PermissionCADelete,
			PermissionCertificateRead,
			PermissionCertificateRequest,
			PermissionCertificateDownload,
			PermissionCertificateVerify,
			PermissionCertificateManage,
			PermissionCertificateRevoke,
			PermissionCertificateDelete,
			PermissionWorkflowRead,
			PermissionWorkflowApprove,
			PermissionWorkflowDelete,
			PermissionCRLRead,
			PermissionCRLManage,
			PermissionAuditRead,
			PermissionAuditManage,
		}
	default:
		return []string{
			PermissionDashboardRead,
			PermissionCARead,
			PermissionCertificateRead,
			PermissionCRLRead,
			PermissionAuditRead,
		}
	}
}
