// Command logwatch 解析 one-api 运行日志，聚合上游/渠道健康度并发现异常（EOF/超时/拒绝/5xx/高延迟）。
//
// 用法：
//
//	# 一次性分析（三种输入源择一）
//	docker logs --since 30m one-api 2>&1 | logwatch              # 管道
//	logwatch -file oneapi.log                                    # 文件
//	logwatch -docker one-api -since 30m                          # 直接抓 docker logs
//
//	# 持续监控（每 interval 抓最近 window 的日志，发现异常打印 ⚠ 告警）
//	logwatch -docker one-api -watch -window 10m -interval 30s
//
//	# JSON 输出 / 自定义阈值 / 接入 CI（有异常退出码 1）
//	logwatch -docker one-api -since 1h -json -conn-errors 5 -err-rate 10 -p95 20000
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"logwatch/core"
)

func main() {
	var (
		file       = flag.String("file", "", "从日志文件读取")
		dockerName = flag.String("docker", "", "直接抓取该容器的 docker logs")
		since      = flag.String("since", "30m", "docker logs 时间窗 (如 30m, 1h, 2026-06-03T09:00:00)")
		watch      = flag.Bool("watch", false, "持续监控模式（需配合 -docker）")
		window     = flag.String("window", "10m", "watch 模式每次分析的回看窗口")
		interval   = flag.Duration("interval", 30*time.Second, "watch 模式刷新间隔")
		jsonOut    = flag.Bool("json", false, "以 JSON 输出报告")
		connErrs   = flag.Int("conn-errors", 0, "单上游连接类错误数告警阈值（0=用默认 3）")
		errRate    = flag.Float64("err-rate", 0, "整体错误率(%)告警阈值（0=用默认 5）")
		p95        = flag.Float64("p95", 0, "p95 延迟(ms)告警阈值（0=用默认 15000）")
	)
	flag.Parse()

	th := core.DefaultThresholds()
	if *connErrs > 0 {
		th.UpstreamConnErrors = *connErrs
	}
	if *errRate > 0 {
		th.ErrorRatePct = *errRate
	}
	if *p95 > 0 {
		th.LatencyP95Ms = *p95
	}

	if *watch {
		if *dockerName == "" {
			fmt.Fprintln(os.Stderr, "错误: -watch 需要配合 -docker <容器名>")
			os.Exit(2)
		}
		runWatch(*dockerName, *window, *interval, th, *jsonOut)
		return
	}

	events, err := loadOnce(*file, *dockerName, *since)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(2)
	}
	r := core.Analyze(events, th)
	emit(r, *jsonOut)
	if r.HasAnomalies() {
		os.Exit(1)
	}
}

func loadOnce(file, dockerName, since string) ([]core.Event, error) {
	switch {
	case file != "":
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return core.ParseReader(f), nil
	case dockerName != "":
		out, err := core.DockerLogsOnce(context.Background(), dockerName, since)
		if err != nil && len(out) == 0 {
			return nil, fmt.Errorf("docker logs 失败: %w", err)
		}
		return core.ParseReader(bytes.NewReader(out)), nil
	default:
		return core.ParseReader(os.Stdin), nil
	}
}

func emit(r core.Report, jsonOut bool) {
	if jsonOut {
		fmt.Println(string(core.ReportToJSON(r)))
		return
	}
	core.WriteReport(os.Stdout, r)
}

func runWatch(container, window string, interval time.Duration, th core.Thresholds, jsonOut bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "logwatch 监控 %s（窗口 %s，每 %s 刷新；Ctrl-C 退出）\n", container, window, interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		out, err := core.DockerLogsOnce(ctx, container, window)
		if err != nil && len(out) == 0 {
			fmt.Fprintf(os.Stderr, "[%s] 抓取日志失败: %v\n", time.Now().Format("15:04:05"), err)
			return
		}
		r := core.Analyze(core.ParseReader(bytes.NewReader(out)), th)
		if jsonOut {
			fmt.Println(string(core.ReportToJSON(r)))
			return
		}
		ts := time.Now().Format("15:04:05")
		if len(r.Anomalies) == 0 {
			fmt.Printf("[%s] ✓ 正常 | 访问 %d 错误 %d | p95 %.0fms | 上游错误 %s\n",
				ts, r.AccessLines, r.Errors, r.LatP95, upstreamBrief(r))
		} else {
			fmt.Printf("[%s] ⚠ 检出 %d 项异常:\n", ts, len(r.Anomalies))
			for _, a := range r.Anomalies {
				fmt.Printf("        - %s\n", a)
			}
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "\n已退出。")
			return
		case <-ticker.C:
			run()
		}
	}
}

func upstreamBrief(r core.Report) string {
	if len(r.Upstreams) == 0 {
		return "无"
	}
	var parts []string
	for _, us := range r.Upstreams {
		parts = append(parts, fmt.Sprintf("%s:%d", us.Host, us.Errors))
	}
	return strings.Join(parts, " ")
}
