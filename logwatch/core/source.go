package core

import (
	"bufio"
	"context"
	"io"
	"os/exec"
)

// ParseReader 从 reader 逐行解析为事件切片。
func ParseReader(r io.Reader) []Event {
	var events []Event
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // 容纳带请求体的长行
	for sc.Scan() {
		if ev, ok := ParseLine(sc.Text()); ok {
			events = append(events, ev)
		}
	}
	return events
}

// DockerLogsOnce 一次性抓取容器在 since 时间窗内的日志（stdout+stderr 合并）。
func DockerLogsOnce(ctx context.Context, container, since string) ([]byte, error) {
	args := []string{"logs"}
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, container)
	cmd := exec.CommandContext(ctx, "docker", args...)
	// docker logs 把应用日志同时写到 stdout/stderr，合并采集。
	out, err := cmd.CombinedOutput()
	return out, err
}
