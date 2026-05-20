package ratio

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const exposedDataTTL = 30 * time.Second

type exposedCache struct {
	data      gin.H
	expiresAt time.Time
}

var (
	exposedData atomic.Value
	rebuildMu   sync.Mutex
)

// InvalidateExposedDataCache 在倍率/价目变更后使对外暴露缓存失效。
func InvalidateExposedDataCache() {
	exposedData.Store((*exposedCache)(nil))
}

func cloneGinH(src gin.H) gin.H {
	dst := make(gin.H, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// GetExposedData 返回对外暴露的倍率/价目快照（带短 TTL 缓存）。
func GetExposedData() gin.H {
	if c, ok := exposedData.Load().(*exposedCache); ok && c != nil && time.Now().Before(c.expiresAt) {
		return cloneGinH(c.data)
	}
	rebuildMu.Lock()
	defer rebuildMu.Unlock()
	if c, ok := exposedData.Load().(*exposedCache); ok && c != nil && time.Now().Before(c.expiresAt) {
		return cloneGinH(c.data)
	}
	newData := gin.H{
		"model_ratio":            GetModelRatioCopy(),
		"completion_ratio":       GetCompletionRatioCopy(),
		"cache_ratio":            GetCacheRatioCopy(),
		"create_cache_ratio":     GetCreateCacheRatioCopy(),
		"model_price":            GetModelPriceCopy(),
		"image_ratio":            GetImageRatioCopy(),
		"audio_ratio":            GetAudioRatioCopy(),
		"audio_completion_ratio": GetAudioCompletionRatioCopy(),
	}
	exposedData.Store(&exposedCache{
		data:      newData,
		expiresAt: time.Now().Add(exposedDataTTL),
	})
	return cloneGinH(newData)
}
