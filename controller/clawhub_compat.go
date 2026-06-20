package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/service"
	"gorm.io/gorm"
)

// ClawHub-compatible registry endpoints backed by the Nacos AI skill registry.
// openclaw-edge speaks the clawhub.ai HTTP API (GET /api/v1/skills/{slug} +
// GET /api/v1/download); pointing its clawhub.registry_url at
// https://<host>/nacos/clawhub lets the fleet install skills from this one-api
// instead of clawhub.ai. Mounted under /nacos/* so it's covered by the existing
// caddy /nacos route and the NacosFeatureGate (open when Nacos is enabled).

func clawhubCompatNamespace(c *gin.Context) string {
	ns := strings.TrimSpace(c.Query("namespaceId"))
	if ns == "" {
		ns = "public"
	}
	return ns
}

// ClawHubCompatSkillDetail serves GET /nacos/clawhub/api/v1/skills/:slug in the
// shape the edge ClawHub client expects: { skill:{slug,...}, latestVersion:{version} }.
func ClawHubCompatSkillDetail(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug required"})
		return
	}
	name, desc, version, err := service.NacosAIClawHubSkillInfo(clawhubCompatNamespace(c), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"skill": gin.H{
			"slug":        name,
			"displayName": name,
			"summary":     desc,
		},
		"latestVersion": gin.H{"version": version},
	})
}

// ClawHubCompatDownload serves GET /nacos/clawhub/api/v1/download?slug=&version=
// returning the skill zip bytes. ClawHub sends `tag` instead of `version` when no
// version is resolved; we map it to a Nacos label.
func ClawHubCompatDownload(c *gin.Context) {
	slug := strings.TrimSpace(c.Query("slug"))
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug required"})
		return
	}
	version := strings.TrimSpace(c.Query("version"))
	label := strings.TrimSpace(c.Query("tag"))
	zipData, err := service.NacosAIGetSkillZIP(clawhubCompatNamespace(c), slug, label, version)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.Data(http.StatusOK, "application/zip", zipData)
}
