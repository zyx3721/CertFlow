package router

import (
	"time"

	"certflow/backend/internal/domain"
)

type errorDocResponse struct {
	Message string `json:"message"`
}

type statusResponse struct {
	Status string `json:"status"`
}

type healthResponse struct {
	OK      bool      `json:"ok"`
	Service string    `json:"service"`
	Time    time.Time `json:"time"`
}

type ocspHealthResponse struct {
	OK      bool `json:"ok"`
	Enabled bool `json:"enabled"`
}

type ocspStatusResponse struct {
	Status       string  `json:"status" example:"good"`
	Valid        bool    `json:"valid"`
	Reason       *string `json:"reason" example:"ocspDisabled"`
	SerialNumber string  `json:"serialNumber"`
	CommonName   string  `json:"commonName,omitempty"`
	RevokedAt    string  `json:"revokedAt,omitempty" example:"2026-07-26 00:12:15"`
	NotBefore    string  `json:"notBefore,omitempty" example:"2026-07-26 00:00:00"`
	NotAfter     string  `json:"notAfter,omitempty" example:"2027-07-26 00:00:00"`
	CheckedAt    string  `json:"checkedAt" example:"2026-07-26 00:59:35"`
}

type loginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"change-me"`
	Provider string `json:"provider" example:"local"`
}

type publicAuthProvidersResponse struct {
	Items                []map[string]any `json:"items"`
	Total                int              `json:"total"`
	PasswordResetEnabled bool             `json:"passwordResetEnabled"`
}

type publicSettingsResponse struct {
	SiteName               string `json:"siteName"`
	LoginName              string `json:"loginName"`
	AppName                string `json:"appName"`
	AppSubtitle            string `json:"appSubtitle"`
	IconData               string `json:"iconData"`
	ExpiryNotificationDays int    `json:"expiryNotificationDays"`
}

type passwordResetVerifyRequest struct {
	Username      string `json:"username"`
	CaptchaToken  string `json:"captchaToken"`
	CaptchaAnswer string `json:"captchaAnswer"`
}

type passwordResetSendRequest struct {
	Username          string `json:"username"`
	VerificationToken string `json:"verificationToken"`
	Channel           string `json:"channel" example:"email"`
	VerifyEmail       string `json:"verifyEmail"`
}

type passwordResetConfirmRequest struct {
	Username          string `json:"username"`
	VerificationToken string `json:"verificationToken"`
	Code              string `json:"code"`
	NewPassword       string `json:"newPassword"`
	ConfirmPassword   string `json:"confirmPassword"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type createCARequest struct {
	Name      string    `json:"name"`
	Type      string    `json:"type" enums:"root,intermediate,issuing"`
	Algorithm string    `json:"algorithm" enums:"RSA-2048,RSA-4096,ECDSA-P256,ECDSA-P384,ED25519"`
	Subject   string    `json:"subject" example:"CN=Example CA,O=Example"`
	ParentID  string    `json:"parentId,omitempty"`
	NotAfter  time.Time `json:"notAfter"`
}

type importCARequest struct {
	Name           string `json:"name" example:"Sunline Root CA"`
	Type           string `json:"type" enums:"root,intermediate,issuing"`
	ParentID       string `json:"parentId,omitempty"`
	CertificatePEM string `json:"certificatePEM"`
	PrivateKeyPEM  string `json:"privateKeyPEM"`
	CSRPEM         string `json:"csrPEM,omitempty"`
}

type certificateRequest struct {
	CAID          string   `json:"caID"`
	CommonName    string   `json:"commonName"`
	Subject       string   `json:"subject"`
	Algorithm     string   `json:"algorithm"`
	Purpose       string   `json:"purpose" enums:"server,client,mtls"`
	SAN           []string `json:"san"`
	ValidityDays  int      `json:"validityDays"`
	CSRPEM        string   `json:"csrPEM,omitempty"`
	PrivateKeyPEM string   `json:"privateKeyPEM,omitempty"`
}

type certificateCSRPreviewRequest struct {
	Subject   string   `json:"subject" example:"CN=api.example.com,O=Acme Corp,C=CN"`
	Algorithm string   `json:"algorithm" enums:"RSA-2048,RSA-4096,ECDSA-P256,ECDSA-P384,ED25519"`
	SAN       []string `json:"san"`
}

type certificateCSRPreviewResponse struct {
	CSRPEM        string `json:"csrPEM"`
	PrivateKeyPEM string `json:"privateKeyPEM"`
}

type certificateCSRInspectRequest struct {
	CSRPEM string `json:"csrPEM"`
}

type certificateCSRInspectResponse struct {
	Subject struct {
		CommonName string `json:"commonName"`
		Org        string `json:"org"`
		OrgUnit    string `json:"orgUnit"`
		Country    string `json:"country"`
		Province   string `json:"province"`
		Locality   string `json:"locality"`
	} `json:"subject"`
	SAN       []string `json:"san"`
	Algorithm string   `json:"algorithm"`
}

type revokeCertificateRequest struct {
	Reason string `json:"reason" enums:"keyCompromise,cACompromise,affiliationChanged,superseded,cessationOfOperation,unspecified"`
}

type workflowUpdateRequest struct {
	Status       string `json:"status" enums:"approved,rejected"`
	RejectReason string `json:"reject_reason,omitempty"`
}

type userStatusRequest struct {
	Disabled bool `json:"disabled"`
}

type roleRequest struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type notificationChannelRequest struct {
	PasswordResetEnabled bool           `json:"passwordResetEnabled"`
	ApprovalEnabled      bool           `json:"approvalEnabled"`
	ClearConfig          bool           `json:"clearConfig"`
	Config               map[string]any `json:"config"`
}

type notificationChannelTestRequest struct {
	To string `json:"to"`
}

type userGroupRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Disabled    bool     `json:"disabled"`
	MemberIDs   []string `json:"memberIds"`
	RoleKeys    []string `json:"roleKeys"`
}

type managedUserRequest struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Password    string   `json:"password,omitempty"`
	RoleKeys    []string `json:"roleKeys"`
	Disabled    bool     `json:"disabled"`
}

type configurationRequest struct {
	Name        string         `json:"name"`
	Enabled     bool           `json:"enabled"`
	ClearConfig bool           `json:"clearConfig"`
	Config      map[string]any `json:"config"`
}

type emailConfigurationRequest struct {
	PasswordResetEnabled bool           `json:"passwordResetEnabled"`
	ClearConfig          bool           `json:"clearConfig"`
	Config               map[string]any `json:"config"`
}

type emailTestRequest struct {
	To string `json:"to" format:"email"`
}

type listResponse struct {
	Items []any `json:"items"`
	Total int   `json:"total"`
}

type caListResponse struct {
	Items []domain.CA `json:"items"`
	Total int         `json:"total"`
}

type caDeletionCheckResponse struct {
	Deletable bool `json:"deletable"`
}

type certificateListResponse struct {
	Items []domain.Certificate `json:"items"`
	Total int                  `json:"total"`
}

type workflowListResponse struct {
	Items []domain.Workflow `json:"items"`
	Total int               `json:"total"`
}

type auditListResponse struct {
	Items []domain.Audit `json:"items"`
	Total int            `json:"total"`
}

type crlListResponse struct {
	Items    []domain.CRLEntry `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}
