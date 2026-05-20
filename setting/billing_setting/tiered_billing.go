package billing_setting

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/pkg/billingexpr"
)

const (
	BillingModeRatio      = "ratio"
	BillingModeTieredExpr = "tiered_expr"
	BillingModeField      = "billing_mode"
	BillingExprField      = "billing_expr"
)

var billingLock sync.RWMutex

var billingModeMap = map[string]string{}
var billingExprMap = map[string]string{}

func UpdateBillingModeByJSONString(jsonStr string) error {
	billingLock.Lock()
	defer billingLock.Unlock()
	billingModeMap = make(map[string]string)
	if jsonStr == "" || jsonStr == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &billingModeMap)
}

func UpdateBillingExprByJSONString(jsonStr string) error {
	billingLock.Lock()
	defer billingLock.Unlock()
	billingExprMap = make(map[string]string)
	if jsonStr == "" || jsonStr == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &billingExprMap)
}

func BillingMode2JSONString() string {
	billingLock.RLock()
	defer billingLock.RUnlock()
	b, _ := json.Marshal(billingModeMap)
	return string(b)
}

func BillingExpr2JSONString() string {
	billingLock.RLock()
	defer billingLock.RUnlock()
	b, _ := json.Marshal(billingExprMap)
	return string(b)
}

func GetBillingMode(model string) string {
	billingLock.RLock()
	defer billingLock.RUnlock()
	if mode, ok := billingModeMap[model]; ok {
		return mode
	}
	return BillingModeRatio
}

func GetBillingExpr(model string) (string, bool) {
	billingLock.RLock()
	defer billingLock.RUnlock()
	expr, ok := billingExprMap[model]
	return expr, ok
}

func GetBillingModeCopy() map[string]string {
	billingLock.RLock()
	defer billingLock.RUnlock()
	out := make(map[string]string, len(billingModeMap))
	for k, v := range billingModeMap {
		out[k] = v
	}
	return out
}

func GetBillingExprCopy() map[string]string {
	billingLock.RLock()
	defer billingLock.RUnlock()
	out := make(map[string]string, len(billingExprMap))
	for k, v := range billingExprMap {
		out[k] = v
	}
	return out
}

func GetPricingSyncData(base map[string]any) map[string]any {
	extra := make(map[string]any, 2)
	if modes := GetBillingModeCopy(); len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if exprs := GetBillingExprCopy(); len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	if base == nil {
		return extra
	}
	for k, v := range extra {
		base[k] = v
	}
	return base
}

func SmokeTestExpr(exprStr string) error {
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
	}
	requests := []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{"anthropic-beta": "fast-mode-2026-02-01"},
			Body:    []byte(`{"service_tier":"fast","stream_options":{"include_usage":true}}`),
		},
	}
	for _, v := range vectors {
		for _, request := range requests {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result %f < 0", v.P, v.C, result)
			}
		}
	}
	return nil
}

func init() {
	if err := UpdateBillingModeByJSONString("{}"); err != nil {
		logger.SysError("billing_setting init mode: " + err.Error())
	}
	if err := UpdateBillingExprByJSONString("{}"); err != nil {
		logger.SysError("billing_setting init expr: " + err.Error())
	}
}
