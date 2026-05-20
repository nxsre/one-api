package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/metrics"
	"github.com/songquanpeng/one-api/relay/model"
)

func ErrorWrapper(err error, code string, statusCode int) *model.ErrorWithStatusCode {
	logger.Error(context.TODO(), fmt.Sprintf("[%s]%+v", code, err))

	if strings.Contains(err.Error(), "broken pipe") || code == "copy_response_body_failed" {
		metrics.ClientDisconnectTotal.WithLabelValues("unknown", "unknown").Inc() // channel_id and model are tricky to get here, will be enhanced if passed
	}

	Error := model.Error{
		Message: err.Error(),
		Type:    "one_api_error",
		Code:    code,
	}
	return &model.ErrorWithStatusCode{
		Error:      Error,
		StatusCode: statusCode,
	}
}
