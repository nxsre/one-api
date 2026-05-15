package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/service"
	"gorm.io/gorm"
)

func nacosCsOperatorFromCtx(c *gin.Context) *service.NacosCsOperator {
	op := &service.NacosCsOperator{}
	if v, ok := c.Get("id"); ok {
		switch t := v.(type) {
		case int:
			op.UserID = t
		case int64:
			op.UserID = int(t)
		case float64:
			op.UserID = int(t)
		}
	}
	if v, ok := c.Get("username"); ok && v != nil {
		op.Username = strings.TrimSpace(fmt.Sprint(v))
	}
	return op
}

// ListNacosCsConfigsAdmin GET /api/nacos/cs/configs
func ListNacosCsConfigsAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	data, err := service.NacosCsList(ns, c.Query("dataId"), c.Query("groupName"), c.Query("search"), page, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// GetNacosCsConfigDetailAdmin GET /api/nacos/cs/configs/detail
func GetNacosCsConfigDetailAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	dataID := strings.TrimSpace(c.Query("dataId"))
	group := strings.TrimSpace(c.Query("groupName"))
	if dataID == "" || group == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "dataId 与 groupName 必填"})
		return
	}
	item, err := service.NacosCsGet(ns, dataID, group)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

type nacosCsPublishBody struct {
	Namespace string `json:"namespace"`
	DataID    string `json:"dataId"`
	GroupName string `json:"groupName"`
	Content   string `json:"content"`
	Type      string `json:"type"`
}

// PublishNacosCsConfigAdmin POST /api/nacos/cs/configs/publish
func PublishNacosCsConfigAdmin(c *gin.Context) {
	var body nacosCsPublishBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	if strings.TrimSpace(body.DataID) == "" || strings.TrimSpace(body.GroupName) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "dataId 与 groupName 必填"})
		return
	}
	op := nacosCsOperatorFromCtx(c)
	created, err := service.NacosCsPublishWithOperator(ns, body.DataID, body.GroupName, body.Content, body.Type, op)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"created": created}})
}

// ListNacosCsConfigHistoryAdmin GET /api/nacos/cs/configs/history
func ListNacosCsConfigHistoryAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	dataID := strings.TrimSpace(c.Query("dataId"))
	group := strings.TrimSpace(c.Query("groupName"))
	if dataID == "" || group == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "dataId 与 groupName 必填"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	data, err := service.NacosCsHistoryList(ns, dataID, group, page, size)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

type nacosCsRollbackBody struct {
	Namespace string `json:"namespace"`
	DataID    string `json:"dataId"`
	GroupName string `json:"groupName"`
	HistoryId int64  `json:"historyId"`
}

// RollbackNacosCsConfigAdmin POST /api/nacos/cs/configs/rollback
func RollbackNacosCsConfigAdmin(c *gin.Context) {
	var body nacosCsRollbackBody
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ns := body.Namespace
	if strings.TrimSpace(ns) == "" {
		ns = "public"
	}
	if body.HistoryId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "historyId 必填"})
		return
	}
	op := nacosCsOperatorFromCtx(c)
	if err := service.NacosCsRollback(ns, body.DataID, body.GroupName, body.HistoryId, op); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteNacosCsConfigAdmin DELETE /api/nacos/cs/configs/item
func DeleteNacosCsConfigAdmin(c *gin.Context) {
	ns := c.DefaultQuery("namespace", "public")
	dataID := strings.TrimSpace(c.Query("dataId"))
	group := strings.TrimSpace(c.Query("groupName"))
	if dataID == "" || group == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "dataId 与 groupName 必填"})
		return
	}
	if err := service.NacosCsDelete(ns, dataID, group); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
