package entity

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

var (
	ErrPasskeyNotFound         = errors.New("passkey credential not found")
	ErrFriendlyPasskeyNotFound = errors.New("Passkey 验证失败，请重试或联系管理员")
)

type PasskeyCredential struct {
	ID              int            `json:"id" gorm:"primaryKey"`
	TenantId        string         `json:"tenant_id" gorm:"type:varchar(36);index;default:'default'"` // Tenant isolation
	UserID          int            `json:"user_id" gorm:"uniqueIndex;not null"`
	CredentialID    string         `json:"credential_id" gorm:"type:varchar(512);uniqueIndex;not null"`
	PublicKey       string         `json:"public_key" gorm:"type:text;not null"`
	AttestationType string         `json:"attestation_type" gorm:"type:varchar(255)"`
	AAGUID          string         `json:"aaguid" gorm:"type:varchar(512)"`
	SignCount       uint32         `json:"sign_count" gorm:"default:0"`
	CloneWarning    bool           `json:"clone_warning"`
	UserPresent     bool           `json:"user_present"`
	UserVerified    bool           `json:"user_verified"`
	BackupEligible  bool           `json:"backup_eligible"`
	BackupState     bool           `json:"backup_state"`
	Transports      string         `json:"transports" gorm:"type:text"`
	Attachment      string         `json:"attachment" gorm:"type:varchar(32)"`
	LastUsedAt      *time.Time     `json:"last_used_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TransportList returns the list of authenticator transports
func (p *PasskeyCredential) TransportList() []protocol.AuthenticatorTransport {
	if p.Transports == "" {
		return nil
	}
	parts := strings.Split(p.Transports, ",")
	transports := make([]protocol.AuthenticatorTransport, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			transports = append(transports, protocol.AuthenticatorTransport(part))
		}
	}
	return transports
}

// SetTransports stores transport list as comma-separated string
func (p *PasskeyCredential) SetTransports(list []protocol.AuthenticatorTransport) {
	strs := make([]string, len(list))
	for i, t := range list {
		strs[i] = string(t)
	}
	p.Transports = strings.Join(strs, ",")
}

// ToWebAuthnCredential converts to webauthn.Credential
func (p *PasskeyCredential) ToWebAuthnCredential() webauthn.Credential {
	credID, _ := base64.RawURLEncoding.DecodeString(p.CredentialID)
	pubKey, _ := base64.RawURLEncoding.DecodeString(p.PublicKey)
	aaguid, _ := base64.RawURLEncoding.DecodeString(p.AAGUID)

	return webauthn.Credential{
		ID:              credID,
		PublicKey:       pubKey,
		AttestationType: p.AttestationType,
		Authenticator: webauthn.Authenticator{
			AAGUID:       aaguid,
			SignCount:    p.SignCount,
			CloneWarning: p.CloneWarning,
			Attachment:   protocol.AuthenticatorAttachment(p.Attachment),
		},
		Transport: p.TransportList(),
		Flags: webauthn.CredentialFlags{
			UserPresent:    p.UserPresent,
			UserVerified:   p.UserVerified,
			BackupEligible: p.BackupEligible,
			BackupState:    p.BackupState,
		},
	}
}

// ApplyValidatedCredential updates fields from a validated credential
func (p *PasskeyCredential) ApplyValidatedCredential(credential *webauthn.Credential) {
	p.SignCount = credential.Authenticator.SignCount
	p.CloneWarning = credential.Authenticator.CloneWarning
	p.BackupState = credential.Flags.BackupState
	now := time.Now()
	p.LastUsedAt = &now
}
