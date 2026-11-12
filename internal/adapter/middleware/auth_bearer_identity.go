package middleware

import (
	"errors"
	"strings"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// fleetDesktopClientID is the public OIDC client_id shared by Switch and Fleet
// desktop (2l-svc-fleet identity.DefaultClientID). Its access token has no
// org_id, so OIDCAuth keeps rejecting it. RootAuth may accept it only when
// the subject is already a local user.
const fleetDesktopClientID = "372642785217478778"

// parseFirstPartyOIDC verifies a Zitadel access token. Tests replace it.
var parseFirstPartyOIDC = func(tokenString string) (*OIDCClaims, error) {
	claims := &OIDCClaims{}
	tok, err := VerifyIDTokenWithJWKS(tokenString, claims)
	if err != nil {
		return nil, err
	}
	if tok == nil || !tok.Valid {
		return nil, errors.New("oidc token not valid")
	}
	return claims, nil
}

// fillFromPlatformOrDesktopBearer resolves a platform HS256 session token or
// a Fleet/Switch desktop OIDC access token. ok is false when neither matches.
func fillFromPlatformOrDesktopBearer(c *gin.Context, bearerToken string) (username string, role, id, status int, ok bool) {
	if accountID, err := common.ValidateIdentitySessionToken(bearerToken); err == nil && accountID > 0 {
		idMapping, _ := common.GetAccountByZitadelSub_ByAccountID(c.Request.Context(), accountID)
		if idMapping != nil && idMapping.ZitadelSub != "" {
			user, _, userErr := repo.GetUserByIDPSubjectAnyTenant(idMapping.ZitadelSub)
			if userErr == nil && user != nil {
				c.Set("identity_account_id", accountID)
				return user.Username, user.Role, user.Id, user.Status, true
			}
		}
	}
	if user := userFromFirstPartyOIDCToken(bearerToken); user != nil {
		return user.Username, user.Role, user.Id, user.Status, true
	}
	return "", 0, 0, 0, false
}

// userFromFirstPartyOIDCToken maps a verified desktop (or newhub-audience)
// access token onto an existing local user. It does not create users.
func userFromFirstPartyOIDCToken(tokenString string) *repo.User {
	if !oidcEnabled || strings.Count(tokenString, ".") != 2 {
		return nil
	}
	claims, err := parseFirstPartyOIDC(tokenString)
	if err != nil || claims == nil || strings.TrimSpace(claims.Subject) == "" {
		return nil
	}
	if !issuerAccepted(claims.Issuer) || !desktopAudienceOK(claims.Audience) {
		return nil
	}
	user, _, err := repo.GetUserByIDPSubjectAnyTenant(claims.Subject)
	if err != nil || user == nil {
		return nil
	}
	return user
}

func issuerAccepted(iss string) bool {
	for _, accepted := range oidcIssuers {
		if iss == accepted {
			return true
		}
	}
	return false
}

func desktopAudienceOK(aud jwt.ClaimStrings) bool {
	for _, a := range aud {
		if a == fleetDesktopClientID {
			return true
		}
	}
	return checkAudience(aud)
}
