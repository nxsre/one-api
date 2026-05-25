package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/model"
)

const channelModelTestTimeout = 30 * time.Second

func runSingleChannelModelTest(ctx context.Context, ch *model.Channel, modelName string) (channelModelTestResult, int64) {
	startedAt := time.Now()
	testCtx, cancel := context.WithTimeout(ctx, channelModelTestTimeout)
	defer cancel()
	msg, detail, testErr, _ := testChannelByModel(testCtx, ch, modelName)
	elapsedMs := time.Since(startedAt).Milliseconds()
	row := buildChannelModelTestResultRow(modelName, ch.Type, msg, testErr, detail)
	row.StartedAt = startedAt.Unix()
	return row, elapsedMs
}

func isChannelModelTestTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "client.timeout exceeded") ||
		strings.Contains(msg, "i/o timeout")
}

func buildChannelModelTestResultRow(modelName string, channelType int, msg string, testErr error, detail channelModelTestHTTPDetail) channelModelTestResult {
	spec := resolveModelTestSpec(modelName, channelType)
	row := channelModelTestResult{
		Model:        modelName,
		TestKind:     modelTestKindLabel(spec.Kind),
		TestProtocol: spec.Protocol,
		Skipped:      spec.Kind == modelTestKindSkip,
	}
	if marshalChannelModelTestDetail(detail) != "" {
		d := detail
		row.Detail = &d
	}
	if testErr != nil {
		row.Success = false
		if isChannelModelTestTimeout(testErr) {
			row.TimedOut = true
			row.Message = fmt.Sprintf("探测超时（%.0f秒）", channelModelTestTimeout.Seconds())
		} else {
			row.Message = testErr.Error()
		}
	} else {
		row.Success = true
		row.Message = msg
	}
	return row
}

func persistChannelModelTestResult(channelID int, baseURL string, row *channelModelTestResult, elapsedMs int64) {
	if channelID <= 0 || row == nil {
		return
	}
	row.ElapsedMs = elapsedMs
	row.Time = float64(elapsedMs) / 1000.0
	if row.StartedAt <= 0 {
		row.StartedAt = time.Now().Unix()
	}
	row.TestedAt = time.Now().Unix()
	detailJSON := ""
	if row.Detail != nil {
		detailJSON = marshalChannelModelTestDetail(*row.Detail)
	}
	_ = model.UpsertChannelModelTestResult(
		channelID,
		baseURL,
		row.Model,
		row.Success,
		row.Skipped,
		row.TimedOut,
		row.TestKind,
		row.TestProtocol,
		row.Message,
		detailJSON,
		row.StartedAt,
		elapsedMs,
	)
}
