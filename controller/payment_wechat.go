package controller

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay"
	wechat "github.com/go-pay/gopay/wechat/v3"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// ---- 配置读取（存于 DB Option，由超管在「设置 - 支付」中维护）----

func payOption(key string) string {
	config.OptionMapRWMutex.RLock()
	v := config.OptionMap[key]
	config.OptionMapRWMutex.RUnlock()
	return strings.TrimSpace(v)
}

func wechatPayEnabled() bool { return payOption("WeChatPayEnabled") == "true" }

// quotaPerYuan 每 1 元（面值）折算的额度；未配置时回落 500000。
func quotaPerYuan() int64 {
	if v, err := strconv.ParseInt(payOption("WeChatPayQuotaPerYuan"), 10, 64); err == nil && v > 0 {
		return v
	}
	return 500000
}

// paymentDiscount 实付折扣（0,1]：实付金额 = 面值 × 折扣。未配置/非法回落 1（不打折）。
// 例：充值面值 200、折扣 0.005 → 实付 1 元；到账额度仍按面值 200 计。
func paymentDiscount() float64 {
	if v, err := strconv.ParseFloat(payOption("WeChatPayDiscount"), 64); err == nil && v > 0 && v <= 1 {
		return v
	}
	return 1
}

// normalizePEM 兼容把私钥粘贴成带字面量 \n 的单行情形。
func normalizePEM(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "\n") && strings.Contains(s, "\\n") {
		s = strings.ReplaceAll(s, "\\n", "\n")
	}
	return s
}

// ---- gopay 微信支付 v3 客户端（按配置指纹缓存，配置变更自动重建）----

var (
	wxClientMu sync.Mutex
	wxClient   *wechat.ClientV3
	wxClientFP string
)

func wechatPayClient() (client *wechat.ClientV3, appid string, err error) {
	mchid := payOption("WeChatPayMchId")
	serial := payOption("WeChatPayCertSerialNo")
	apiV3 := payOption("WeChatPayApiV3Key")
	privKey := normalizePEM(payOption("WeChatPayPrivateKey"))
	appid = payOption("WeChatPayAppId")
	if mchid == "" || serial == "" || apiV3 == "" || privKey == "" || appid == "" {
		return nil, "", errors.New("微信支付未完整配置（appid/商户号/证书序列号/APIv3密钥/商户私钥）")
	}
	sum := sha256.Sum256([]byte(mchid + "|" + serial + "|" + apiV3 + "|" + privKey + "|" + appid))
	fp := hex.EncodeToString(sum[:])

	wxClientMu.Lock()
	defer wxClientMu.Unlock()
	if wxClient != nil && wxClientFP == fp {
		return wxClient, appid, nil
	}
	c, e := wechat.NewClientV3(mchid, serial, apiV3, privKey)
	if e != nil {
		return nil, "", fmt.Errorf("初始化微信支付客户端失败: %w", e)
	}
	// 自动下载并周期刷新微信平台证书，用于响应/回调验签。
	if e = c.AutoVerifySign(); e != nil {
		return nil, "", fmt.Errorf("加载微信平台证书失败: %w", e)
	}
	wxClient = c
	wxClientFP = fp
	return wxClient, appid, nil
}

func genOrderNo() string {
	var b [6]byte
	_, _ = crand.Read(b[:])
	return "wx" + time.Now().Format("20060102150405") + hex.EncodeToString(b[:])
}

func containsStr(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func payFail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
}

// ---- 用户侧：创建微信扫码（Native）订单 ----

type createWeChatNativeRequest struct {
	Amount int `json:"amount"` // 充值金额（元，正整数）
}

func CreateWeChatNativeOrder(c *gin.Context) {
	if !wechatPayEnabled() {
		payFail(c, "微信支付未启用")
		return
	}
	userId := c.GetInt(ctxkey.Id)
	allowed, _ := model.ResolveUserAllowedPaymentChannels(userId)
	if !containsStr(allowed, "wxpay") {
		payFail(c, "当前账号未开通微信支付")
		return
	}

	var req createWeChatNativeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payFail(c, err.Error())
		return
	}
	if req.Amount <= 0 || req.Amount > 100000 {
		payFail(c, "充值金额无效（应为 1 ~ 100000 元的整数）")
		return
	}

	client, appid, err := wechatPayClient()
	if err != nil {
		payFail(c, err.Error())
		return
	}

	faceCents := req.Amount * 100
	quota := int64(req.Amount) * quotaPerYuan()
	// 实付 = 面值 × 折扣（向上取整到分，且不低于微信最小 1 分）。
	amountCents := int(math.Ceil(float64(faceCents) * paymentDiscount()))
	if amountCents < 1 {
		amountCents = 1
	}
	orderNo := genOrderNo()
	order := &model.PaymentOrder{
		OrderNo:     orderNo,
		UserId:      userId,
		Channel:     "wxpay",
		TradeType:   "NATIVE",
		FaceCents:   faceCents,
		AmountCents: amountCents,
		Quota:       quota,
		Status:      "pending",
	}
	if err = model.CreatePaymentOrder(order); err != nil {
		payFail(c, "创建订单失败: "+err.Error())
		return
	}

	notifyURL := strings.TrimRight(payOption("WeChatPayNotifyDomain"), "/") + "/api/pay/wechat/notify"
	bm := make(gopay.BodyMap)
	bm.Set("appid", appid).
		Set("mchid", payOption("WeChatPayMchId")).
		Set("description", "额度充值").
		Set("out_trade_no", orderNo).
		Set("notify_url", notifyURL)
	bm.SetBodyMap("amount", func(b gopay.BodyMap) {
		b.Set("total", amountCents).Set("currency", "CNY")
	})

	wxRsp, err := client.V3TransactionNative(c.Request.Context(), bm)
	if err != nil {
		payFail(c, "微信下单失败: "+err.Error())
		return
	}
	if wxRsp.Code != wechat.Success || wxRsp.Response == nil {
		msg := wxRsp.Error
		if msg == "" {
			msg = wxRsp.ErrResponse.Message
		}
		payFail(c, "微信下单被拒绝: "+msg)
		return
	}
	codeURL := wxRsp.Response.CodeUrl
	_ = model.UpdatePaymentOrderCodeURL(orderNo, codeURL)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"order_no":   orderNo,
			"code_url":   codeURL,
			"amount":     req.Amount,                  // 面值（元）
			"pay_amount": float64(amountCents) / 100.0, // 实付（元，已打折）
			"discount":   paymentDiscount(),
			"quota":      quota,
		},
	})
}

// ---- 用户侧：查询订单状态（带主动查单兜底，防回调丢失）----

func GetPaymentOrderStatus(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	orderNo := strings.TrimSpace(c.Param("order_no"))
	o, err := model.GetPaymentOrderByNo(orderNo)
	if err != nil {
		payFail(c, "订单不存在")
		return
	}
	if o.UserId != userId && !model.IsAdmin(userId) {
		payFail(c, "无权查看该订单")
		return
	}

	if o.Status == "pending" {
		if client, _, cerr := wechatPayClient(); cerr == nil {
			if rsp, qerr := client.V3TransactionQueryOrder(c.Request.Context(), wechat.OutTradeNo, orderNo); qerr == nil &&
				rsp.Code == wechat.Success && rsp.Response != nil && rsp.Response.TradeState == "SUCCESS" {
				total := 0
				if rsp.Response.Amount != nil {
					total = rsp.Response.Amount.Total
				}
				if credited, uid, q, serr := model.SettlePaymentOrderPaid(orderNo, rsp.Response.TransactionId, total); serr == nil {
					if credited {
						model.RecordTopupLog(c.Request.Context(), uid, "微信扫码支付充值", int(q))
					}
					o.Status = "paid"
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"order_no": o.OrderNo,
			"status":   o.Status,
			"amount":   o.AmountCents / 100,
			"quota":    o.Quota,
		},
	})
}

// ---- 微信异步回调（公开，无需登录；靠平台证书验签 + APIv3 解密）----

type wxNotifyResource struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionId string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	Amount        struct {
		Total      int `json:"total"`
		PayerTotal int `json:"payer_total"`
	} `json:"amount"`
}

func wxNotifyReply(c *gin.Context, httpCode int, code, message string) {
	c.JSON(httpCode, gin.H{"code": code, "message": message})
}

func WeChatPayNotifyHandler(c *gin.Context) {
	client, _, err := wechatPayClient()
	if err != nil {
		logger.SysError("wechat notify: client init failed: " + err.Error())
		wxNotifyReply(c, http.StatusInternalServerError, "FAIL", "支付未配置")
		return
	}
	notifyReq, err := wechat.V3ParseNotify(c.Request)
	if err != nil {
		wxNotifyReply(c, http.StatusBadRequest, "FAIL", "解析回调失败")
		return
	}
	if err = notifyReq.VerifySignByPKMap(client.WxPublicKeyMap()); err != nil {
		logger.SysError("wechat notify: verify sign failed: " + err.Error())
		wxNotifyReply(c, http.StatusUnauthorized, "FAIL", "验签失败")
		return
	}
	var res wxNotifyResource
	if err = notifyReq.DecryptCipherTextToStruct(payOption("WeChatPayApiV3Key"), &res); err != nil {
		logger.SysError("wechat notify: decrypt failed: " + err.Error())
		wxNotifyReply(c, http.StatusBadRequest, "FAIL", "解密失败")
		return
	}

	if res.TradeState == "SUCCESS" {
		credited, uid, q, serr := model.SettlePaymentOrderPaid(res.OutTradeNo, res.TransactionId, res.Amount.Total)
		if serr != nil {
			// 入账失败（如金额不符/DB 错误）：返回非 200 让微信重试。
			logger.SysError(fmt.Sprintf("wechat notify settle failed for %s: %v", res.OutTradeNo, serr))
			wxNotifyReply(c, http.StatusInternalServerError, "FAIL", "处理失败")
			return
		}
		if credited {
			model.RecordTopupLog(c.Request.Context(), uid, "微信扫码支付充值", int(q))
		}
	}
	wxNotifyReply(c, http.StatusOK, "SUCCESS", "成功")
}
