package middleware

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"

	"github.com/golang-jwt/jwt/v5"
)

func withDesktopOIDC(t *testing.T, subject string) {
	t.Helper()
	prevEnabled := oidcEnabled
	prevIssuers := oidcIssuers
	prevParse := parseFirstPartyOIDC
	oidcEnabled = true
	oidcIssuers = []string{"https://identity.lurus.cn"}
	parseFirstPartyOIDC = func(string) (*OIDCClaims, error) {
		return &OIDCClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:   "https://identity.lurus.cn",
				Subject:  subject,
				Audience: jwt.ClaimStrings{fleetDesktopClientID},
			},
		}, nil
	}
	t.Cleanup(func() {
		oidcEnabled = prevEnabled
		oidcIssuers = prevIssuers
		parseFirstPartyOIDC = prevParse
	})
}

func seedMappedUser(t *testing.T, name, subject string, role int) {
	t.Helper()
	user := &repo.User{
		Username: name, Role: role, Status: common.UserStatusEnabled,
		Email: name + "@local", TenantId: "default",
	}
	if err := repo.DB.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	mapping := &entity.UserIdentityMapping{
		LurusUserID: user.Id, IDPSubject: subject, TenantID: "default", IsActive: true,
	}
	if err := repo.DB.Create(mapping).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}
}

func TestRootAuth_DesktopOIDCTokenMapsExistingRoot(t *testing.T) {
	_, cleanup := setupCoverDB(t)
	defer cleanup()
	withDesktopOIDC(t, "sub-root")
	seedMappedUser(t, "rootdesk", "sub-root", common.RoleRootUser)

	r := buildRoleRouter(map[string]interface{}{}, RootAuth)
	w := serveRole(r, map[string]string{"Authorization": "Bearer aaa.bbb.ccc"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRootAuth_DesktopOIDCTokenRejectsNonRoot(t *testing.T) {
	_, cleanup := setupCoverDB(t)
	defer cleanup()
	withDesktopOIDC(t, "sub-user")
	seedMappedUser(t, "plaindesk", "sub-user", common.RoleCommonUser)

	r := buildRoleRouter(map[string]interface{}{}, RootAuth)
	w := serveRole(r, map[string]string{"Authorization": "Bearer aaa.bbb.ccc"})
	if strings.Contains(w.Body.String(), `"ok":true`) {
		t.Fatalf("non-root reached handler; body=%s", w.Body.String())
	}
}

func TestRootAuth_DesktopOIDCTokenRejectsWrongAudience(t *testing.T) {
	_, cleanup := setupCoverDB(t)
	defer cleanup()
	withDesktopOIDC(t, "sub-root")
	parseFirstPartyOIDC = func(string) (*OIDCClaims, error) {
		return &OIDCClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:   "https://identity.lurus.cn",
				Subject:  "sub-root",
				Audience: jwt.ClaimStrings{"some-other-client"},
			},
		}, nil
	}
	seedMappedUser(t, "rootdesk2", "sub-root", common.RoleRootUser)

	r := buildRoleRouter(map[string]interface{}{}, RootAuth)
	w := serveRole(r, map[string]string{"Authorization": "Bearer aaa.bbb.ccc"})
	if strings.Contains(w.Body.String(), `"ok":true`) {
		t.Fatalf("wrong audience reached handler; body=%s", w.Body.String())
	}
}

func TestRootAuth_DesktopOIDCTokenMatchesBoundAccount(t *testing.T) {
	_, cleanup := setupCoverDB(t)
	defer cleanup()
	withDesktopOIDC(t, "sub-bound")
	prev := accountByIDPSubject
	accountByIDPSubject = func(context.Context, string) (*common.IdentityMapping, error) {
		return &common.IdentityMapping{ID: 42}, nil
	}
	t.Cleanup(func() { accountByIDPSubject = prev })
	acct := int64(42)
	user := &repo.User{
		Username: "boundroot", Role: common.RoleRootUser, Status: common.UserStatusEnabled,
		Email: "bound@local", TenantId: "default", LurusAccountID: &acct,
	}
	if err := repo.DB.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	r := buildRoleRouter(map[string]interface{}{}, RootAuth)
	w := serveRole(r, map[string]string{"Authorization": "Bearer aaa.bbb.ccc"})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ok":true`) {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}
