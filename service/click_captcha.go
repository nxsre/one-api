package service

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/golang/freetype/truetype"
	"github.com/google/uuid"
	captcha "github.com/wenlng/go-captcha/v2"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/click"
	goredis "github.com/go-redis/redis/v8"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	loginClickCaptchaVerifyPadding = 10
	loginCaptchaRedisKeyPrefix     = "one-api:login:captcha:"
	loginCaptchaRedisTTL           = 10 * time.Minute
)

// LoginCaptchaClickPoint 用户在主图上的点击坐标（与生成图像像素坐标一致）
type LoginCaptchaClickPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type loginCaptchaStoredDot struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

var (
	clickCaptchaMu       sync.Mutex
	clickCaptchaInstance click.Captcha
	clickCaptchaInitErr  error
)

func loginCaptchaRedisKey(id string) string {
	return loginCaptchaRedisKeyPrefix + id
}

func loginCaptchaCharPool() []string {
	const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	out := make([]string, 0, len(charset))
	for _, r := range charset {
		out = append(out, string(r))
	}
	return out
}

func loginCaptchaBackground(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			base := uint8(rand.Intn(50) + 160)
			img.SetRGBA(x, y, color.RGBA{
				R: base - uint8(rand.Intn(25)),
				G: base - uint8(rand.Intn(20)),
				B: base + uint8(rand.Intn(20)),
				A: 255,
			})
		}
	}
	return img
}

func initLoginClickCaptcha() (click.Captcha, error) {
	fontFace, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse bundled font: %w", err)
	}

	builder := captcha.NewClickBuilder(
		click.WithRangeLen(option.RangeVal{Min: 5, Max: 6}),
		click.WithRangeVerifyLen(option.RangeVal{Min: 3, Max: 4}),
	)
	bg := loginCaptchaBackground(300, 220)
	builder.SetResources(
		click.WithChars(loginCaptchaCharPool()),
		click.WithFonts([]*truetype.Font{fontFace}),
		click.WithBackgrounds([]image.Image{bg}),
	)
	return builder.Make(), nil
}

func getLoginClickCaptcha() (click.Captcha, error) {
	clickCaptchaMu.Lock()
	defer clickCaptchaMu.Unlock()
	if clickCaptchaInitErr != nil {
		return nil, clickCaptchaInitErr
	}
	if clickCaptchaInstance != nil {
		return clickCaptchaInstance, nil
	}
	capt, err := initLoginClickCaptcha()
	if err != nil {
		clickCaptchaInitErr = err
		return nil, err
	}
	clickCaptchaInstance = capt
	return capt, nil
}

// GenerateLoginClickCaptcha 生成点击验证码。若 Redis 可用则答案仅存于 Redis 并返回 captchaID；
// 否则返回 legacyDotsJSON 供写入 cookie session（单机/无 Redis）。
func GenerateLoginClickCaptcha() (masterImageBase64, thumbImageBase64 string, dotNum int, captchaID string, legacyDotsJSON []byte, err error) {
	capt, err := getLoginClickCaptcha()
	if err != nil {
		return "", "", 0, "", nil, err
	}
	data, err := capt.Generate()
	if err != nil {
		return "", "", 0, "", nil, err
	}
	masterB64, err := data.GetMasterImage().ToBase64Data()
	if err != nil {
		return "", "", 0, "", nil, err
	}
	thumbB64, err := data.GetThumbImage().ToBase64Data()
	if err != nil {
		return "", "", 0, "", nil, err
	}

	dotsMap := data.GetData()
	if len(dotsMap) == 0 {
		return "", "", 0, "", nil, fmt.Errorf("empty captcha dots")
	}
	ordered := make([]loginCaptchaStoredDot, len(dotsMap))
	for i := 0; i < len(dotsMap); i++ {
		d := dotsMap[i]
		if d == nil {
			return "", "", 0, "", nil, fmt.Errorf("missing dot index %d", i)
		}
		ordered[i] = loginCaptchaStoredDot{X: d.X, Y: d.Y, Width: d.Width, Height: d.Height}
	}
	raw, err := json.Marshal(ordered)
	if err != nil {
		return "", "", 0, "", nil, err
	}

	if common.RedisEnabled && common.RDB != nil {
		id := uuid.New().String()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := common.RDB.Set(ctx, loginCaptchaRedisKey(id), raw, loginCaptchaRedisTTL).Err(); err != nil {
			return "", "", 0, "", nil, err
		}
		return masterB64, thumbB64, len(ordered), id, nil, nil
	}
	return masterB64, thumbB64, len(ordered), "", raw, nil
}

// ValidateLoginClickCaptcha 校验用户点击序列是否与会话中保存的点区域匹配（顺序一致）。
func ValidateLoginClickCaptcha(storedJSON []byte, points []LoginCaptchaClickPoint) bool {
	var expected []loginCaptchaStoredDot
	if err := json.Unmarshal(storedJSON, &expected); err != nil {
		return false
	}
	if len(points) != len(expected) {
		return false
	}
	for i := range expected {
		pt := points[i]
		e := expected[i]
		if !click.Validate(pt.X, pt.Y, e.X, e.Y, e.Width, e.Height, loginClickCaptchaVerifyPadding) {
			return false
		}
	}
	return true
}

// ConsumeLoginClickCaptchaRedis 校验并消费一次登录点击验证码。成功后删除 Redis 键，防止重放。
func ConsumeLoginClickCaptchaRedis(clientCaptchaID, sessionPendingID string, points []LoginCaptchaClickPoint) bool {
	if clientCaptchaID == "" || sessionPendingID == "" || clientCaptchaID != sessionPendingID {
		return false
	}
	if _, err := uuid.Parse(clientCaptchaID); err != nil {
		return false
	}
	if common.RDB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := loginCaptchaRedisKey(clientCaptchaID)
	raw, err := common.RDB.Get(ctx, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return false
		}
		return false
	}
	if !ValidateLoginClickCaptcha([]byte(raw), points) {
		return false
	}
	if err := common.RDB.Del(ctx, key).Err(); err != nil {
		return false
	}
	return true
}
