package controller

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

func GetAllLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	includeErrors := c.Query("include_errors") == "true" || c.Query("include_errors") == "1"
	logs, total, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId, includeErrors)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    pageInfo,
	})
}

func GetUserLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt(ctxkey.Id)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	includeErrors := c.Query("include_errors") == "true" || c.Query("include_errors") == "1"
	logs, total, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId, includeErrors)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    pageInfo,
	})
}

func GetLogByKey(c *gin.Context) {
	tokenId := c.GetInt(ctxkey.TokenId)
	if tokenId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}
	logs, err := model.GetLogByTokenId(tokenId)
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
		"data":    logs,
	})
}

// SearchAllLogs 已废弃（避免误用模糊搜索）。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// SearchUserLogs 已废弃。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

func GetLogsStat(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
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
		"data": gin.H{
			"quota": stat.Quota,
			"rpm":   stat.Rpm,
			"tpm":   stat.Tpm,
		},
	})
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString(ctxkey.Username)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
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
		"data": gin.H{
			"quota": stat.Quota,
			"rpm":   stat.Rpm,
			"tpm":   stat.Tpm,
		},
	})
}

// GetBillingSummary 按账期/时区聚合账单，逐项透出 token 明细并折算 USD 金额。
// 维度通过 group_by（逗号分隔：channel/model/token/group）控制，with_day=1 时按自然日拆分。
func GetBillingSummary(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tzOffset, _ := strconv.Atoi(c.Query("tz_offset"))
	if c.Query("tz_offset") == "" {
		tzOffset = config.DefaultBillingTzOffsetMinutes
	}
	// period=current 时按账期起始日(BillingCycleDayOfMonth)自动切出当前账期区间。
	if c.Query("period") == "current" || (startTimestamp == 0 && endTimestamp == 0) {
		startTimestamp, endTimestamp = model.CurrentBillingPeriod(tzOffset)
	}
	withDay := c.Query("with_day") == "true" || c.Query("with_day") == "1"
	var groupBy []string
	if g := strings.TrimSpace(c.Query("group_by")); g != "" {
		groupBy = strings.Split(g, ",")
	}
	rows, err := model.SummarizeBilling(model.BillingSummaryParams{
		StartTimestamp:  startTimestamp,
		EndTimestamp:    endTimestamp,
		TzOffsetMinutes: tzOffset,
		WithDay:         withDay,
		GroupBy:         groupBy,
		ModelName:       c.Query("model_name"),
		Username:        c.Query("username"),
		TokenName:       c.Query("token_name"),
		Channel:         func() int { v, _ := strconv.Atoi(c.Query("channel")); return v }(),
		Group:           c.Query("group"),
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"tz_offset":       tzOffset,
			"start_timestamp": startTimestamp,
			"end_timestamp":   endTimestamp,
			"cycle_day":       config.BillingCycleDayOfMonth,
			"settlement_days": config.BillingSettlementDays,
			"rows":            rows,
		},
	})
}

// ExportLogs 以 CSV 流式导出消费日志（含全部 token 明细列 + 折算 USD 金额 + 本地时区时间）。
// 复用 GetAllLogs 的筛选参数，用 offset 分页游标遍历，绕过列表查询的条数上限。
func ExportLogs(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	includeErrors := c.Query("include_errors") == "true" || c.Query("include_errors") == "1"
	tzOffset, _ := strconv.Atoi(c.Query("tz_offset"))
	if c.Query("tz_offset") == "" {
		tzOffset = config.DefaultBillingTzOffsetMinutes
	}
	loc := time.FixedZone("billing", tzOffset*60)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"logs_export.csv\"")
	// UTF-8 BOM，便于 Excel 正确识别中文。
	_, _ = c.Writer.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(c.Writer)
	defer w.Flush()
	_ = w.Write([]string{
		"time", "request_id", "user_id", "username", "token_name", "channel_id", "channel_name",
		"group", "model_name", "is_stream", "prompt_tokens", "completion_tokens",
		"cached_tokens", "cache_creation_tokens", "reasoning_tokens",
		"image_prompt_tokens", "audio_prompt_tokens", "audio_completion_tokens",
		"quota", "amount_usd", "elapsed_ms", "content",
	})

	const pageSize = 1000
	for startIdx := 0; ; startIdx += pageSize {
		logs, _, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, startIdx, pageSize, channel, group, requestId, includeErrors)
		if err != nil {
			// 头已发出，无法再改状态码，记录后中断。
			_ = w.Write([]string{"export error: " + err.Error()})
			return
		}
		for _, lg := range logs {
			_ = w.Write([]string{
				time.Unix(lg.CreatedAt, 0).In(loc).Format("2006-01-02 15:04:05"),
				lg.RequestId,
				lg.UserPublicID,
				lg.Username,
				lg.TokenName,
				strconv.Itoa(lg.ChannelId),
				lg.ChannelName,
				lg.Group,
				lg.ModelName,
				strconv.FormatBool(lg.IsStream),
				strconv.Itoa(lg.PromptTokens),
				strconv.Itoa(lg.CompletionTokens),
				strconv.Itoa(lg.CachedTokens),
				strconv.Itoa(lg.CacheCreationTokens),
				strconv.Itoa(lg.ReasoningTokens),
				strconv.Itoa(lg.ImagePromptTokens),
				strconv.Itoa(lg.AudioPromptTokens),
				strconv.Itoa(lg.AudioCompletionTokens),
				strconv.Itoa(lg.Quota),
				fmt.Sprintf("%.6f", float64(lg.Quota)/config.QuotaPerUnit),
				strconv.FormatInt(lg.ElapsedTime, 10),
				lg.Content,
			})
		}
		w.Flush()
		if len(logs) < pageSize {
			break
		}
	}
}

func DeleteHistoryLogs(c *gin.Context) {
	targetTimestamp, _ := strconv.ParseInt(c.Query("target_timestamp"), 10, 64)
	if targetTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "target timestamp is required",
		})
		return
	}
	count, err := model.DeleteOldLog(targetTimestamp)
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
		"data":    count,
	})
}
