package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"

	imagesAsset "github.com/wenlng/go-captcha-assets/resources/images"
	tilesAsset "github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/rotate"
	"github.com/wenlng/go-captcha/v2/slide"

	"github.com/google/uuid"
)

// Multi-mode login captcha built on github.com/wenlng/go-captcha. Click mode
// reuses the self-contained generator in click_captcha.go; slide and rotate
// pull bundled image assets from go-captcha-assets. The chosen mode is stored
// alongside its answer so validation can dispatch correctly.

const (
	loginSlideVerifyPadding  = 8
	loginRotateVerifyPadding = 10
)

// loginCaptchaStored is the answer persisted in Redis/session for one challenge.
type loginCaptchaStored struct {
	Mode  string                  `json:"mode"`
	Dots  []loginCaptchaStoredDot `json:"dots,omitempty"`
	X     int                     `json:"x,omitempty"`
	Y     int                     `json:"y,omitempty"`
	Angle int                     `json:"angle,omitempty"`
}

// LoginCaptchaAnswer is the user-submitted solution for any mode.
type LoginCaptchaAnswer struct {
	Mode  string                   `json:"mode"`
	Dots  []LoginCaptchaClickPoint `json:"dots,omitempty"`
	X     int                      `json:"x,omitempty"`
	Y     int                      `json:"y,omitempty"`
	Angle int                      `json:"angle,omitempty"`
}

// LoginCaptchaChallenge is the public payload returned to the browser. Only the
// fields relevant to Mode are populated.
type LoginCaptchaChallenge struct {
	Mode        string
	MasterImage string // full data URI
	ThumbImage  string // full data URI (click thumb / slide tile / rotate thumb)
	DotNum      int    // click
	TileX       int    // slide: tile initial X
	TileY       int    // slide: tile initial Y
	TileWidth   int    // slide
	TileHeight  int    // slide
	ThumbSize   int    // rotate: thumb square size

	storedJSON []byte
}

var (
	slideCaptchaMu       sync.Mutex
	slideCaptchaInstance slide.Captcha
	slideCaptchaInitErr  error

	rotateCaptchaMu       sync.Mutex
	rotateCaptchaInstance rotate.Captcha
	rotateCaptchaInitErr  error
)

func getSlideCaptcha() (slide.Captcha, error) {
	slideCaptchaMu.Lock()
	defer slideCaptchaMu.Unlock()
	if slideCaptchaInitErr != nil {
		return nil, slideCaptchaInitErr
	}
	if slideCaptchaInstance != nil {
		return slideCaptchaInstance, nil
	}
	tiles, err := tilesAsset.GetTiles()
	if err != nil {
		slideCaptchaInitErr = err
		return nil, err
	}
	graphs := make([]*slide.GraphImage, 0, len(tiles))
	for _, t := range tiles {
		graphs = append(graphs, &slide.GraphImage{
			OverlayImage: t.OverlayImage,
			ShadowImage:  t.ShadowImage,
			MaskImage:    t.MaskImage,
		})
	}
	bgs, err := imagesAsset.GetImages()
	if err != nil {
		slideCaptchaInitErr = err
		return nil, err
	}
	builder := slide.NewBuilder()
	builder.SetResources(
		slide.WithGraphImages(graphs),
		slide.WithBackgrounds(bgs),
	)
	slideCaptchaInstance = builder.Make()
	return slideCaptchaInstance, nil
}

func getRotateCaptcha() (rotate.Captcha, error) {
	rotateCaptchaMu.Lock()
	defer rotateCaptchaMu.Unlock()
	if rotateCaptchaInitErr != nil {
		return nil, rotateCaptchaInitErr
	}
	if rotateCaptchaInstance != nil {
		return rotateCaptchaInstance, nil
	}
	imgs, err := imagesAsset.GetImages()
	if err != nil {
		rotateCaptchaInitErr = err
		return nil, err
	}
	builder := rotate.NewBuilder()
	builder.SetResources(rotate.WithImages(imgs))
	rotateCaptchaInstance = builder.Make()
	return rotateCaptchaInstance, nil
}

// GenerateLoginCaptcha builds a challenge for the given concrete mode (click,
// slide or rotate). Returns the challenge plus the answer JSON to persist.
func GenerateLoginCaptcha(mode string) (*LoginCaptchaChallenge, error) {
	switch mode {
	case config.LoginCaptchaModeSlide:
		return generateSlideChallenge()
	case config.LoginCaptchaModeRotate:
		return generateRotateChallenge()
	default:
		return generateClickChallenge()
	}
}

func generateClickChallenge() (*LoginCaptchaChallenge, error) {
	master, thumb, dotNum, dotsJSON, err := generateClickRaw()
	if err != nil {
		return nil, err
	}
	stored := loginCaptchaStored{Mode: config.LoginCaptchaModeClick}
	if err := json.Unmarshal(dotsJSON, &stored.Dots); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	return &LoginCaptchaChallenge{
		Mode:        config.LoginCaptchaModeClick,
		MasterImage: "data:image/jpeg;base64," + master,
		ThumbImage:  "data:image/png;base64," + thumb,
		DotNum:      dotNum,
		storedJSON:  raw,
	}, nil
}

func generateSlideChallenge() (*LoginCaptchaChallenge, error) {
	capt, err := getSlideCaptcha()
	if err != nil {
		return nil, err
	}
	data, err := capt.Generate()
	if err != nil {
		return nil, err
	}
	block := data.GetData()
	if block == nil {
		return nil, fmt.Errorf("empty slide block")
	}
	masterB64, err := data.GetMasterImage().ToBase64Data()
	if err != nil {
		return nil, err
	}
	tileB64, err := data.GetTileImage().ToBase64Data()
	if err != nil {
		return nil, err
	}
	stored := loginCaptchaStored{Mode: config.LoginCaptchaModeSlide, X: block.X, Y: block.Y}
	raw, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	return &LoginCaptchaChallenge{
		Mode:        config.LoginCaptchaModeSlide,
		MasterImage: "data:image/jpeg;base64," + masterB64,
		ThumbImage:  "data:image/png;base64," + tileB64,
		TileX:       block.DX,
		TileY:       block.DY,
		TileWidth:   block.Width,
		TileHeight:  block.Height,
		storedJSON:  raw,
	}, nil
}

func generateRotateChallenge() (*LoginCaptchaChallenge, error) {
	capt, err := getRotateCaptcha()
	if err != nil {
		return nil, err
	}
	data, err := capt.Generate()
	if err != nil {
		return nil, err
	}
	block := data.GetData()
	if block == nil {
		return nil, fmt.Errorf("empty rotate block")
	}
	masterB64, err := data.GetMasterImage().ToBase64Data()
	if err != nil {
		return nil, err
	}
	thumbB64, err := data.GetThumbImage().ToBase64Data()
	if err != nil {
		return nil, err
	}
	stored := loginCaptchaStored{Mode: config.LoginCaptchaModeRotate, Angle: block.Angle}
	raw, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	return &LoginCaptchaChallenge{
		Mode:        config.LoginCaptchaModeRotate,
		MasterImage: "data:image/png;base64," + masterB64,
		ThumbImage:  "data:image/png;base64," + thumbB64,
		ThumbSize:   block.Width,
		storedJSON:  raw,
	}, nil
}

// StoredJSON exposes the answer payload to persist for this challenge.
func (c *LoginCaptchaChallenge) StoredJSON() []byte { return c.storedJSON }

// validateLoginCaptchaPayload dispatches validation by the stored mode.
func validateLoginCaptchaPayload(storedJSON []byte, answer LoginCaptchaAnswer) bool {
	var stored loginCaptchaStored
	if err := json.Unmarshal(storedJSON, &stored); err != nil {
		// Backward-compat: legacy click challenges stored a bare dots array.
		var dots []loginCaptchaStoredDot
		if err2 := json.Unmarshal(storedJSON, &dots); err2 != nil {
			return false
		}
		stored = loginCaptchaStored{Mode: config.LoginCaptchaModeClick, Dots: dots}
	}
	switch stored.Mode {
	case config.LoginCaptchaModeSlide:
		return slide.Validate(answer.X, answer.Y, stored.X, stored.Y, loginSlideVerifyPadding)
	case config.LoginCaptchaModeRotate:
		return rotate.Validate(answer.Angle, stored.Angle, loginRotateVerifyPadding)
	default:
		dotsJSON, err := json.Marshal(stored.Dots)
		if err != nil {
			return false
		}
		return ValidateLoginClickCaptcha(dotsJSON, answer.Dots)
	}
}

// ValidateLoginCaptchaSession validates against a session-stored answer (no Redis).
func ValidateLoginCaptchaSession(storedJSON []byte, answer LoginCaptchaAnswer) bool {
	return validateLoginCaptchaPayload(storedJSON, answer)
}

// ConsumeLoginCaptchaRedis validates and one-shot-consumes a Redis-stored answer.
func ConsumeLoginCaptchaRedis(clientCaptchaID, sessionPendingID string, answer LoginCaptchaAnswer) bool {
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
		return false
	}
	if !validateLoginCaptchaPayload([]byte(raw), answer) {
		return false
	}
	if err := common.RDB.Del(ctx, key).Err(); err != nil {
		return false
	}
	return true
}

// StoreLoginCaptchaRedis persists the answer JSON and returns the challenge id.
func StoreLoginCaptchaRedis(storedJSON []byte) (string, error) {
	if common.RDB == nil {
		return "", fmt.Errorf("redis unavailable")
	}
	id := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := common.RDB.Set(ctx, loginCaptchaRedisKey(id), storedJSON, loginCaptchaRedisTTL).Err(); err != nil {
		return "", err
	}
	return id, nil
}
