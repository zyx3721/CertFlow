package pki

import (
	"testing"

	"certflow/backend/internal/domain"
)

func TestShouldNotifyApprovalSkipsApplicantsWhoCanApprove(t *testing.T) {
	tests := []struct {
		name string
		user domain.User
		want bool
	}{
		{name: "standard applicant", user: domain.User{Permissions: []string{domain.PermissionCertificateRequest}}, want: true},
		{name: "approver applicant", user: domain.User{Permissions: []string{domain.PermissionWorkflowApprove}}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldNotifyApproval(test.user); got != test.want {
				t.Fatalf("shouldNotifyApproval() = %t, want %t", got, test.want)
			}
		})
	}
}
