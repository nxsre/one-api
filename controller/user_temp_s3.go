package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

// UserS3Enable 启用并生成 AK/SK（Secret 仅在本次响应中返回）。
func UserS3Enable(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "站点未开启 S3（请在 root 管理员「系统设置」中开启 S3 兼容存储）"})
		return
	}
	id := c.GetInt(ctxkey.Id)
	ak, sk, err := model.UserEnableS3(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"access_key": ak,
			"secret_key": sk,
			"region":     common.S3Region,
		},
	})
}

// UserS3Disable 关闭并失效当前密钥（不删除磁盘对象）。
func UserS3Disable(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "站点未开启 S3"})
		return
	}
	id := c.GetInt(ctxkey.Id)
	if err := model.UserDisableS3(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// UserS3RegenerateSecret 保留 Access Key，仅更换 Secret。
func UserS3RegenerateSecret(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "站点未开启 S3"})
		return
	}
	id := c.GetInt(ctxkey.Id)
	sk, err := model.UserRegenerateS3Secret(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"secret_key": sk,
			"region":     common.S3Region,
		},
	})
}

// UserS3RotateKeys 同时更换 AK 与 SK，原密钥立即失效。
func UserS3RotateKeys(c *gin.Context) {
	if !common.S3SiteOpen() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "站点未开启 S3"})
		return
	}
	id := c.GetInt(ctxkey.Id)
	ak, sk, err := model.UserS3RotateKeys(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"access_key": ak,
			"secret_key": sk,
			"region":     common.S3Region,
		},
	})
}
