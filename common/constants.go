package common

import "time"

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.0"            // this hard coding will be replaced automatically when building, no need to manually change
// BuildID 单次编译构建号（通常为 UTC 时间戳），由 go build -ldflags "-X ...BuildID=..." 注入。
var BuildID = "dev"
