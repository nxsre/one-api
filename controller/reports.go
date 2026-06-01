package controller

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

// PlatformGetBillingReport GET /platform/reports/billing
func PlatformGetBillingReport(c *gin.Context) {
	start, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)

	if start == 0 {
		start = time.Now().AddDate(0, -1, 0).Unix() // Default to last month
	}
	if end == 0 {
		end = time.Now().Unix()
	}

	rows, err := model.GetPlatformBillingReport(start, end)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

// PlatformExportBillingReport GET /platform/reports/billing/export
func PlatformExportBillingReport(c *gin.Context) {
	start, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)

	if start == 0 {
		start = time.Now().AddDate(0, -1, 0).Unix()
	}
	if end == 0 {
		end = time.Now().Unix()
	}

	rows, err := model.GetPlatformBillingExport(start, end)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment;filename=platform_billing_report.csv")
	_, _ = c.Writer.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{
		"Tenant ID", "Tenant Name", "Quota", "Amount USD", "Request Count",
		"Prompt Tokens", "Completion Tokens", "Cached Tokens", "Cache Creation Tokens",
		"Reasoning Tokens", "Image Prompt Tokens", "Audio Prompt Tokens", "Audio Completion Tokens",
	})

	for _, r := range rows {
		writer.Write([]string{
			fmt.Sprintf("%d", r.TenantId),
			r.TenantName,
			fmt.Sprintf("%d", r.Quota),
			fmt.Sprintf("%.6f", r.AmountUSD),
			fmt.Sprintf("%d", r.RequestCount),
			fmt.Sprintf("%d", r.PromptTokens),
			fmt.Sprintf("%d", r.CompletionTokens),
			fmt.Sprintf("%d", r.CachedTokens),
			fmt.Sprintf("%d", r.CacheCreationTokens),
			fmt.Sprintf("%d", r.ReasoningTokens),
			fmt.Sprintf("%d", r.ImagePromptTokens),
			fmt.Sprintf("%d", r.AudioPromptTokens),
			fmt.Sprintf("%d", r.AudioCompletionTokens),
		})
	}
	writer.Flush()
}

// TenantGetBillingReport GET /tenant_console/reports/billing
func TenantGetBillingReport(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_billing")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	start, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)

	if start == 0 {
		start = time.Now().AddDate(0, -1, 0).Unix()
	}
	if end == 0 {
		end = time.Now().Unix()
	}

	rows, err := model.GetTenantBillingReport(tid, start, end)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

// TenantExportBillingReport GET /tenant_console/reports/billing/export
func TenantExportBillingReport(c *gin.Context) {
	tid, err := GetTenantConsoleTenantID(c, "manage_billing")
	if err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}

	start, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)

	if start == 0 {
		start = time.Now().AddDate(0, -1, 0).Unix()
	}
	if end == 0 {
		end = time.Now().Unix()
	}

	rows, err := model.GetTenantBillingExport(tid, start, end)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment;filename=tenant_billing_report.csv")
	_, _ = c.Writer.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{
		"User ID", "Username", "Channel ID", "Channel Name", "Model Name", "Quota", "Amount USD",
		"Request Count", "Prompt Tokens", "Completion Tokens", "Cached Tokens", "Cache Creation Tokens",
		"Reasoning Tokens", "Image Prompt Tokens", "Audio Prompt Tokens", "Audio Completion Tokens",
	})

	for _, r := range rows {
		writer.Write([]string{
			fmt.Sprintf("%d", r.UserId),
			r.Username,
			fmt.Sprintf("%d", r.ChannelId),
			r.ChannelName,
			r.ModelName,
			fmt.Sprintf("%d", r.Quota),
			fmt.Sprintf("%.6f", r.AmountUSD),
			fmt.Sprintf("%d", r.RequestCount),
			fmt.Sprintf("%d", r.PromptTokens),
			fmt.Sprintf("%d", r.CompletionTokens),
			fmt.Sprintf("%d", r.CachedTokens),
			fmt.Sprintf("%d", r.CacheCreationTokens),
			fmt.Sprintf("%d", r.ReasoningTokens),
			fmt.Sprintf("%d", r.ImagePromptTokens),
			fmt.Sprintf("%d", r.AudioPromptTokens),
			fmt.Sprintf("%d", r.AudioCompletionTokens),
		})
	}
	writer.Flush()
}
