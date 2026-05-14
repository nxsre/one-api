package router

import (
	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/controller"
)

// SetS3CompatRouter 注册与 S3 path-style 子集兼容的临时对象路由（需环境变量启用）。
func SetS3CompatRouter(r *gin.Engine) {
	r.OPTIONS("/s3/:bucket", controller.S3OptionsPrefixed)
	r.HEAD("/s3/:bucket", controller.S3BucketHeadPrefixed)
	r.PUT("/s3/:bucket", controller.S3BucketWritePrefixed)
	r.DELETE("/s3/:bucket", controller.S3BucketDeletePrefixed)
	r.GET("/s3/:bucket", controller.S3BucketGETPrefixed)
	r.Any("/s3/:bucket/*key", controller.S3ObjectPrefixed)
}

// SetS3PathStyleRootRouter 在站点根路径注册 /{bucket}/…（与 AWS CLI path-style + 自定义 endpoint 一致）。
// 必须在 SetRelayRouter 之后调用，且需 S3_ADDRESSING=path。
func SetS3PathStyleRootRouter(r *gin.Engine) {
	if !common.S3Enabled || !common.S3PathStyleAtRoot {
		return
	}
	r.OPTIONS("/:bucket", controller.S3PathStyleOPTIONS)
	r.HEAD("/:bucket", controller.S3PathStyleBucketHead)
	r.PUT("/:bucket", controller.S3PathStyleBucketWrite)
	r.DELETE("/:bucket", controller.S3PathStyleBucketDelete)
	r.GET("/:bucket", controller.S3PathStyleBucketGET)
	r.Any("/:bucket/*objKey", controller.S3PathStyleObject)
}
