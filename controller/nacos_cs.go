package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/service"
	"gorm.io/gorm"
)

// NacosCsConfigListV3 GET /nacos/v3/admin/cs/config/list
func NacosCsConfigListV3(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	data, err := service.NacosCsList(service.NormalizeNacosNamespaceID(c.Query("namespaceId")),
		c.Query("dataId"), c.Query("groupName"), c.Query("search"), pageNo, pageSize)
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, data)
}

// NacosCsConfigListV1 GET /nacos/v1/cs/configs — 响应体为列表 JSON（无 v3 包络），与 nacos-cli 一致。
func NacosCsConfigListV1(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	ns := c.Query("tenant")
	if ns == "" {
		ns = "public"
	}
	data, err := service.NacosCsList(service.NormalizeNacosNamespaceID(ns),
		c.Query("dataId"), c.Query("group"), c.Query("search"), pageNo, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// nacosCsClientWantsCiphertext 是否对客户端 GET 返回密文：配置项默认 + 查询串覆盖（plain=1 强制明文；cipher=1 强制密文）。
func nacosCsClientWantsCiphertext(c *gin.Context) bool {
	if env.ParseBoolLoose(c.Query("plain"), false) || c.Query("plain") == "1" {
		return false
	}
	if q := strings.TrimSpace(c.Query("cipher")); q != "" {
		return env.ParseBoolLoose(q, false)
	}
	return config.NacosCsClientGetReturnCiphertext
}

// NacosCsConfigGet GET …/v3/console/cs/config（控制台/管理端）— 始终解密为明文。
func NacosCsConfigGet(c *gin.Context) {
	nacosCsConfigGetImpl(c, false)
}

// NacosCsConfigGetClient GET /nacos/v3/client/cs/config — 可按配置或查询参数返回密文（模拟 SDK）。
func NacosCsConfigGetClient(c *gin.Context) {
	nacosCsConfigGetImpl(c, nacosCsClientWantsCiphertext(c))
}

func nacosCsConfigGetImpl(c *gin.Context, deliverCiphertext bool) {
	dataID := c.Query("dataId")
	group := c.Query("groupName")
	if dataID == "" || group == "" {
		nacosV3Err(c, 10000, "dataId 与 groupName 必填")
		return
	}
	ns := service.NormalizeNacosNamespaceID(c.Query("namespaceId"))
	item, err := service.NacosCsGetForClient(ns, dataID, group, deliverCiphertext)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "config 不存在")
			return
		}
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, item)
}

// NacosCsConfigPublish POST /nacos/v3/admin/cs/config
func NacosCsConfigPublish(c *gin.Context) {
	dataID := c.PostForm("dataId")
	group := c.PostForm("groupName")
	content := c.PostForm("content")
	if dataID == "" || group == "" {
		nacosV3Err(c, 10000, "dataId 与 groupName 必填")
		return
	}
	ok, err := service.NacosCsPublish(service.NormalizeNacosNamespaceID(c.PostForm("namespaceId")), dataID, group, content, c.PostForm("type"))
	if err != nil {
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, ok)
}

// NacosCsConfigHistoryListV3 GET /nacos/v3/admin/cs/config/history
func NacosCsConfigHistoryListV3(c *gin.Context) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	dataID := strings.TrimSpace(c.Query("dataId"))
	group := strings.TrimSpace(c.Query("groupName"))
	if dataID == "" || group == "" {
		nacosV3Err(c, 10000, "dataId 与 groupName 必填")
		return
	}
	data, err := service.NacosCsHistoryList(service.NormalizeNacosNamespaceID(c.Query("namespaceId")), dataID, group, pageNo, pageSize)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nacosV3Err(c, 20004, "config 不存在")
			return
		}
		nacosV3Err(c, 30000, err.Error())
		return
	}
	nacosV3OK(c, data)
}

type nacosCsRollbackV3Body struct {
	DataID    string `json:"dataId"`
	GroupName string `json:"groupName"`
	HistoryId int64  `json:"historyId"`
}

// NacosCsConfigRollbackV3 POST /nacos/v3/admin/cs/config/rollback
func NacosCsConfigRollbackV3(c *gin.Context) {
	var body nacosCsRollbackV3Body
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 20002, err.Error())
		return
	}
	if body.HistoryId <= 0 || strings.TrimSpace(body.DataID) == "" || strings.TrimSpace(body.GroupName) == "" {
		nacosV3Err(c, 10000, "dataId、groupName、historyId 必填")
		return
	}
	if err := service.NacosCsRollback(service.NormalizeNacosNamespaceID(c.Query("namespaceId")), body.DataID, body.GroupName, body.HistoryId, nil); err != nil {
		nacosV3Err(c, 20002, err.Error())
		return
	}
	nacosV3OK(c, true)
}
