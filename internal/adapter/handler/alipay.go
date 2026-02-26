package handler

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/smartwalle/alipay/v3"
)

var (
	alipayClient     *alipay.Client
	alipayClientOnce sync.Once
	alipayClientErr  error
)

func getAlipayClient() (*alipay.Client, error) {
	alipayClientOnce.Do(func() {
		appId := common.AlipayAppId
		privateKey := os.Getenv("ALIPAY_PRIVATE_KEY")
		if appId == "" || privateKey == "" {
			alipayClientErr = errors.New("alipay app_id or private_key not configured")
			return
		}
		isProduction := os.Getenv("ALIPAY_SANDBOX") != "true"
		alipayClient, alipayClientErr = alipay.New(appId, privateKey, isProduction)
		if alipayClientErr != nil {
			return
		}
		// Load alipay public key for signature verification
		alipayPublicKey := os.Getenv("ALIPAY_PUBLIC_KEY")
		if alipayPublicKey != "" {
			alipayClientErr = alipayClient.LoadAliPayPublicKey(alipayPublicKey)
		}
	})
	return alipayClient, alipayClientErr
}

// resetAlipayClient allows re-initialization when config changes
func resetAlipayClient() {
	alipayClientOnce = sync.Once{}
	alipayClient = nil
	alipayClientErr = nil
}

func getAlipayUserInfoByCode(ctx context.Context, authCode string) (userId string, nickname string, err error) {
	if authCode == "" {
		return "", "", errors.New("auth_code is empty")
	}

	client, err := getAlipayClient()
	if err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Exchange auth_code for access_token
	tokenReq := alipay.SystemOauthToken{
		GrantType: "authorization_code",
		Code:      authCode,
	}
	tokenResp, err := client.SystemOauthToken(ctx, tokenReq)
	if err != nil {
		return "", "", errors.New("failed to get alipay access token: " + err.Error())
	}
	if tokenResp.UserId == "" {
		return "", "", errors.New("alipay returned empty user_id")
	}

	// Get user info
	userInfoReq := alipay.UserInfoShare{
		AuthToken: tokenResp.AccessToken,
	}
	userInfoResp, err := client.UserInfoShare(ctx, userInfoReq)
	if err != nil {
		// If user info fails, still return user_id (some apps may not have user_info permission)
		return tokenResp.UserId, "", nil
	}

	return tokenResp.UserId, userInfoResp.NickName, nil
}

func AlipayOAuth(c *gin.Context) {
	session := sessions.Default(c)
	state := c.Query("state")
	if state == "" || session.Get("oauth_state") == nil || state != session.Get("oauth_state").(string) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "state is empty or not same",
		})
		return
	}
	username := session.Get("username")
	if username != nil {
		AlipayBind(c)
		return
	}

	if !common.AlipayOAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过支付宝登录以及注册",
		})
		return
	}

	authCode := c.Query("code")
	if authCode == "" {
		authCode = c.Query("auth_code")
	}
	alipayUserId, nickname, err := getAlipayUserInfoByCode(c.Request.Context(), authCode)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	user := repo.User{
		AlipayId: alipayUserId,
	}

	if repo.IsAlipayIdAlreadyTaken(user.AlipayId) {
		err := user.FillUserByAlipayId()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		if user.Id == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户已注销",
			})
			return
		}
	} else {
		if common.RegisterEnabled {
			user.Username = common.AlipayUsernamePrefix + strconv.Itoa(repo.GetMaxUserId()+1)
			if nickname != "" {
				user.DisplayName = nickname
			} else {
				user.DisplayName = "Alipay User"
			}
			user.Role = common.RoleCommonUser
			user.Status = common.UserStatusEnabled
			user.TenantId = "default"

			affCode := session.Get("aff")
			inviterId := 0
			if affCode != nil {
				inviterId, _ = repo.GetUserIdByAffCode(affCode.(string))
			}

			if err := user.Insert(inviterId); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}
	}

	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}
	setupLogin(&user, c)
}

func AlipayBind(c *gin.Context) {
	if !common.AlipayOAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过支付宝登录以及注册",
		})
		return
	}

	// Check session first before making any API calls (security best practice)
	session := sessions.Default(c)
	id := session.Get("id")
	userId, ok := id.(int)
	if !ok || userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录或会话已过期",
		})
		return
	}

	authCode := c.Query("code")
	if authCode == "" {
		authCode = c.Query("auth_code")
	}
	alipayUserId, _, err := getAlipayUserInfoByCode(c.Request.Context(), authCode)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	user := repo.User{
		AlipayId: alipayUserId,
	}
	if repo.IsAlipayIdAlreadyTaken(user.AlipayId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该支付宝账户已被绑定",
		})
		return
	}

	user.Id = userId
	err = user.FillUserById()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user.AlipayId = alipayUserId
	err = user.Update(false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "bind",
	})
}
