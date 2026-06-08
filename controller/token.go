package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/agentpolicy"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/network"
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/model"
	"net/http"
	"strconv"
)

// tokenScope 描述当前请求者可操作的令牌范围：
//   - platformAdmin 为 true：平台管理员，可管理所有用户的令牌；
//   - tenantID > 0：租户管理员，仅可管理本租户全部成员的令牌；
//   - 两者皆否：普通用户，仅可管理自己的令牌。
type tokenScope struct {
	platformAdmin bool
	tenantID      int
}

// resolveTokenScope 依据当前用户角色解析其令牌操作范围。
// 平台管理员语义与前端 isAdmin() 一致：租户管理员（RoleTenantAdmin）不视为平台管理员，
// 而是被限定在本租户范围内。
func resolveTokenScope(c *gin.Context) tokenScope {
	role := c.GetInt(ctxkey.Role)
	if role >= model.RoleAdminUser && role != model.RoleTenantAdmin {
		return tokenScope{platformAdmin: true}
	}
	if role == model.RoleTenantAdmin {
		if tid := model.GetUserTenantIDNumeric(c.GetInt(ctxkey.Id)); tid > 0 {
			return tokenScope{tenantID: tid}
		}
	}
	return tokenScope{}
}

// loadTokenForScope 在「按 id 操作单个令牌」的场景下，按当前范围加载令牌并完成归属校验：
//   - 平台管理员：仅按 id 加载；
//   - 租户管理员：按 id 加载后，校验令牌所属用户确属本租户；
//   - 普通用户：按 (id, userId) 加载，天然限定为本人令牌。
func loadTokenForScope(scope tokenScope, id int, userId int) (*model.Token, error) {
	if scope.platformAdmin {
		return model.GetTokenById(id)
	}
	if scope.tenantID > 0 {
		token, err := model.GetTokenById(id)
		if err != nil {
			return nil, err
		}
		if model.GetUserTenantIDNumeric(token.UserId) != scope.tenantID {
			return nil, fmt.Errorf("只能管理本租户下成员的令牌")
		}
		return token, nil
	}
	return model.GetTokenByIds(id, userId)
}

// enrichTokensWithUsername 批量回填令牌的创建人（所属用户）用户名，供前端展示。
func enrichTokensWithUsername(tokens []*model.Token) {
	if len(tokens) == 0 {
		return
	}
	idSet := make(map[int]struct{}, len(tokens))
	for _, token := range tokens {
		idSet[token.UserId] = struct{}{}
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	nameMap := model.GetUsernamesByIds(ids)
	for _, token := range tokens {
		token.Username = nameMap[token.UserId]
	}
}

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}

	order := c.Query("order")
	scope := resolveTokenScope(c)
	var tokens []*model.Token
	var err error
	switch {
	case scope.platformAdmin:
		tokens, err = model.GetAllTokensForAdmin(p*config.ItemsPerPage, config.ItemsPerPage, order)
	case scope.tenantID > 0:
		tokens, err = model.GetTenantTokens(scope.tenantID, p*config.ItemsPerPage, config.ItemsPerPage, order)
	default:
		tokens, err = model.GetAllUserTokens(userId, p*config.ItemsPerPage, config.ItemsPerPage, order)
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	enrichTokensWithUsername(tokens)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tokens,
	})
	return
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	keyword := c.Query("keyword")
	scope := resolveTokenScope(c)
	var tokens []*model.Token
	var err error
	switch {
	case scope.platformAdmin:
		tokens, err = model.SearchAllTokens(keyword)
	case scope.tenantID > 0:
		tokens, err = model.SearchTenantTokens(scope.tenantID, keyword)
	default:
		tokens, err = model.SearchUserTokens(userId, keyword)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	enrichTokensWithUsername(tokens)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tokens,
	})
	return
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	token, err := loadTokenForScope(resolveTokenScope(c), id, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    token,
	})
	return
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt(ctxkey.TokenId)
	userId := c.GetInt(ctxkey.Id)
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   token.RemainQuota,
		"total_used":      0, // not supported currently
		"total_available": token.RemainQuota,
		"expires_at":      expiredAt * 1000,
	})
}

func validateToken(c *gin.Context, token model.Token) error {
	if len(token.Name) > 30 {
		return fmt.Errorf("令牌名称过长")
	}
	if token.Subnet != nil && *token.Subnet != "" {
		err := network.IsValidSubnets(*token.Subnet)
		if err != nil {
			return fmt.Errorf("无效的网段：%s", err.Error())
		}
	}
	return nil
}

func AddToken(c *gin.Context) {
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = validateToken(c, token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("参数错误：%s", err.Error()),
		})
		return
	}

	cleanToken := model.Token{
		UserId:            c.GetInt(ctxkey.Id),
		Name:              token.Name,
		Key:               random.GenerateKey(),
		CreatedTime:       helper.GetTimestamp(),
		AccessedTime:      helper.GetTimestamp(),
		ExpiredTime:       token.ExpiredTime,
		RemainQuota:       token.RemainQuota,
		UnlimitedQuota:    token.UnlimitedQuota,
		Models:            token.Models,
		Group:             token.Group,
		Subnet:            token.Subnet,
		AgentClientPolicy: token.AgentClientPolicy,
	}
	if cleanToken.AgentClientPolicy != nil && !cleanToken.AgentClientPolicy.IsZero() {
		agentpolicy.SetEnabled(true)
	}
	err = cleanToken.Insert()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanToken,
	})
	return
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	scope := resolveTokenScope(c)
	var err error
	if scope.platformAdmin || scope.tenantID > 0 {
		// 先按范围校验归属（租户管理员限定本租户），再按 id 删除。
		if _, err = loadTokenForScope(scope, id, userId); err == nil {
			err = model.DeleteTokenByIdOnly(id)
		}
	} else {
		err = model.DeleteTokenById(id, userId)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	statusOnly := c.Query("status_only")
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = validateToken(c, token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("参数错误：%s", err.Error()),
		})
		return
	}
	cleanToken, err := loadTokenForScope(resolveTokenScope(c), token.Id, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if token.Status == model.TokenStatusEnabled {
		if cleanToken.Status == model.TokenStatusExpired && cleanToken.ExpiredTime <= helper.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "令牌已过期，无法启用，请先修改令牌过期时间，或者设置为永不过期",
			})
			return
		}
		if cleanToken.Status == model.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "令牌可用额度已用尽，无法启用，请先修改令牌剩余额度，或者设置为无限额度",
			})
			return
		}
	}
	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		// If you add more fields, please also update token.Update()
		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.Models = token.Models
		cleanToken.Subnet = token.Subnet
		cleanToken.Group = token.Group
		cleanToken.AgentClientPolicy = token.AgentClientPolicy
		if cleanToken.AgentClientPolicy != nil && !cleanToken.AgentClientPolicy.IsZero() {
			agentpolicy.SetEnabled(true)
		}
	}
	err = cleanToken.Update()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanToken,
	})
	return
}
