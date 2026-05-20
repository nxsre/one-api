package service

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/cfg"
	"github.com/songquanpeng/one-api/model"
)

var clusterHeartbeatOnce sync.Once

// StartNacosConsoleClusterSelfHeartbeat 启动 one-api 自身作为 Nacos 控制台集群节点的注册与心跳上报。
// 同时启动过期节点清扫任务（将长时间未上报的节点标记为 DOWN）。
func StartNacosConsoleClusterSelfHeartbeat(port int) {
	clusterHeartbeatOnce.Do(func() {
		go runSelfClusterHeartbeat(port)
		go runClusterNodeStaleSweep()
	})
}

func runSelfClusterHeartbeat(port int) {
	address := getClusterAdvertiseAddress(port)
	registerOrUpdateClusterNode(address, "UP")

	interval := time.Duration(cfg.GetNacosConsoleClusterHeartbeatInterval()) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		registerOrUpdateClusterNode(address, "UP")
	}
}

func getClusterAdvertiseAddress(port int) string {
	if adv := cfg.GetNacosConsoleClusterAdvertiseAddr(); adv != "" {
		return adv
	}
	return "127.0.0.1:" + strconv.Itoa(port)
}

func registerOrUpdateClusterNode(address, state string) {
	ip, portStr, err := net.SplitHostPort(address)
	if err != nil {
		// 兜底解析
		parts := strings.Split(address, ":")
		ip = parts[0]
		if len(parts) > 1 {
			portStr = parts[1]
		} else {
			portStr = "0"
		}
	}
	port, _ := strconv.Atoi(portStr)
	now := time.Now()

	var node model.NacosConsoleClusterNode
	if err := model.DB.Where("address = ?", address).First(&node).Error; err == nil {
		node.Ip = ip
		node.Port = port
		node.State = state
		node.UpdatedAt = now
		model.DB.Save(&node)
	} else {
		node = model.NacosConsoleClusterNode{
			Address:   address,
			Ip:        ip,
			Port:      port,
			State:     state,
			UpdatedAt: now,
		}
		model.DB.Create(&node)
	}
}

func runClusterNodeStaleSweep() {
	interval := time.Duration(cfg.GetNacosConsoleClusterStaleSweepInterval()) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		sweepStaleClusterNodes()
	}
}

func sweepStaleClusterNodes() {
	ttlSec := cfg.GetNacosConsoleClusterStaleTTL()
	cutoff := time.Now().Add(-time.Duration(ttlSec) * time.Second)
	model.DB.Model(&model.NacosConsoleClusterNode{}).
		Where("updated_at < ? AND state <> ?", cutoff, "DOWN").
		Update("state", "DOWN")
}
