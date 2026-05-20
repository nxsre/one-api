package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/requestaudit"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func recordRelayErrorLog(c *gin.Context, bizErr *relaymodel.ErrorWithStatusCode, startAt time.Time) {
	if c == nil || bizErr == nil || bizErr.StatusCode == -1 {
		return
	}
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		return
	}
	useTime := int(time.Since(startAt).Seconds())
	if useTime <= 0 {
		useTime = 1
	}
	modelName := strings.TrimSpace(c.GetString(ctxkey.OriginalModel))
	if modelName == "" {
		modelName = strings.TrimSpace(c.GetString(ctxkey.RequestModel))
	}
	msg := strings.TrimSpace(bizErr.Error.Message)
	if msg == "" {
		msg = "relay failed"
	}
	content := fmt.Sprintf("[%d] %s", bizErr.StatusCode, msg)
	if code := bizErr.Error.Code; code != nil && fmt.Sprint(code) != "" {
		content = fmt.Sprintf("[%d|%v] %s", bizErr.StatusCode, code, msg)
	}
	model.RecordErrorLog(c.Request.Context(), &model.Log{
		UserId:      userId,
		Content:     content,
		ModelName:   modelName,
		TokenName:   c.GetString(ctxkey.TokenName),
		TokenId:     c.GetInt(ctxkey.TokenId),
		ChannelId:   c.GetInt(ctxkey.ChannelId),
		Group:       c.GetString(ctxkey.Group),
		UseTime:     useTime,
		ElapsedTime: time.Since(startAt).Milliseconds(),
		Other:       requestaudit.ConsumeLogOtherJSON(c),
	})
}
