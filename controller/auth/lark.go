package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/model"
)

// 飞书开放平台 OAuth 2.0（v2 token + authen user_info），响应均为 { code, msg, data } 包裹。
const (
	larkOAuthTokenURL    = "https://open.feishu.cn/open-apis/authen/v2/oauth/token"
	larkOAuthUserInfoURL = "https://open.feishu.cn/open-apis/authen/v1/user_info"
)

type larkTokenData struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

type larkTokenResponse struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data larkTokenData `json:"data"`
}

type larkUserData struct {
	Name   string `json:"name"`
	OpenID string `json:"open_id"`
	UnionID string `json:"union_id"`
}

type larkUserInfoResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data larkUserData `json:"data"`
}

func getLarkUserInfoByCode(code string) (*larkUserData, error) {
	if code == "" {
		return nil, errors.New("无效的参数")
	}
	if config.LarkClientId == "" || config.LarkClientSecret == "" {
		return nil, errors.New("飞书 OAuth 未配置完整（App ID / App Secret）")
	}
	body := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     config.LarkClientId,
		"client_secret": config.LarkClientSecret,
		"code":          code,
		"redirect_uri":  fmt.Sprintf("%s/oauth/lark", strings.TrimRight(strings.TrimSpace(config.ServerAddress), "/")),
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, larkOAuthTokenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	httpClient := client.NewOutboundHTTPClient(15 * time.Second)
	res, err := httpClient.Do(req)
	if err != nil {
		logger.SysLog(err.Error())
		return nil, errors.New("无法连接至飞书服务器，请稍后重试！")
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var tok larkTokenResponse
	if err := json.Unmarshal(raw, &tok); err != nil {
		return nil, fmt.Errorf("解析飞书 token 响应失败: %w", err)
	}
	if tok.Code != 0 {
		if tok.Msg != "" {
			return nil, fmt.Errorf("飞书授权失败: %s (code=%d)", tok.Msg, tok.Code)
		}
		return nil, fmt.Errorf("飞书授权失败: code=%d", tok.Code)
	}
	if tok.Data.AccessToken == "" {
		return nil, errors.New("飞书未返回 access_token，请检查重定向地址是否与控制台一致")
	}

	req2, err := http.NewRequest(http.MethodGet, larkOAuthUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Authorization", "Bearer "+tok.Data.AccessToken)
	req2.Header.Set("Accept", "application/json")

	res2, err := httpClient.Do(req2)
	if err != nil {
		logger.SysLog(err.Error())
		return nil, errors.New("无法连接至飞书服务器，请稍后重试！")
	}
	defer res2.Body.Close()
	raw2, err := io.ReadAll(res2.Body)
	if err != nil {
		return nil, err
	}
	var userWrap larkUserInfoResponse
	if err := json.Unmarshal(raw2, &userWrap); err != nil {
		return nil, fmt.Errorf("解析飞书用户信息失败: %w", err)
	}
	if userWrap.Code != 0 {
		if userWrap.Msg != "" {
			return nil, fmt.Errorf("飞书用户信息失败: %s (code=%d)", userWrap.Msg, userWrap.Code)
		}
		return nil, fmt.Errorf("飞书用户信息失败: code=%d", userWrap.Code)
	}
	if userWrap.Data.OpenID == "" {
		return nil, errors.New("飞书未返回 open_id，请确认应用已开通网页登录并申请 authen.user_info 权限")
	}
	return &userWrap.Data, nil
}

func LarkOAuth(c *gin.Context) {
	ctx := c.Request.Context()
	if !config.LarkOAuthEnabled || config.LarkClientId == "" || config.LarkClientSecret == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "飞书登录未启用或未配置完整",
		})
		return
	}
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
		LarkBind(c)
		return
	}
	code := c.Query("code")
	larkUser, err := getLarkUserInfoByCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user := model.User{
		LarkId: larkUser.OpenID,
	}
	if model.IsLarkIdAlreadyTaken(user.LarkId) {
		err := user.FillUserByLarkId()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	} else {
		if config.RegisterEnabled {
			user.Username = "lark_" + strconv.Itoa(model.GetMaxUserId()+1)
			if larkUser.Name != "" {
				user.DisplayName = larkUser.Name
			} else {
				user.DisplayName = "飞书用户"
			}
			user.Role = model.RoleCommonUser
			user.Status = model.UserStatusEnabled

			if err := user.Insert(ctx, 0); err != nil {
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

	if user.Status != model.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}
	controller.SetupLogin(&user, c)
}

func LarkBind(c *gin.Context) {
	if !config.LarkOAuthEnabled || config.LarkClientId == "" || config.LarkClientSecret == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "飞书登录未启用或未配置完整",
		})
		return
	}
	code := c.Query("code")
	larkUser, err := getLarkUserInfoByCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user := model.User{
		LarkId: larkUser.OpenID,
	}
	if model.IsLarkIdAlreadyTaken(user.LarkId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该飞书账户已被绑定",
		})
		return
	}
	session := sessions.Default(c)
	id := session.Get("id")
	user.Id = id.(int)
	err = user.FillUserById()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user.LarkId = larkUser.OpenID
	err = user.Update(false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "bind",
	})
}
