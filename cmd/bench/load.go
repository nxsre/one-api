package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type headerList []string

func (h *headerList) String() string { return strings.Join(*h, ", ") }
func (h *headerList) Set(v string) error {
	*h = append(*h, v)
	return nil
}

type loadConfig struct {
	url         string
	method      string
	body        []byte
	headers     headerList
	concurrency int
	rps         int
	duration    time.Duration
	total       int64
	timeout     time.Duration
	stream      bool
}

func runLoadCmd(args []string) {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	var cfg loadConfig
	fs.StringVar(&cfg.url, "url", "", "target URL (required)")
	fs.StringVar(&cfg.method, "method", "", "HTTP method (default GET, or POST when -body is set)")
	bodyArg := fs.String("body", "", "request body string, or @file to read from a file")
	fs.Var(&cfg.headers, "H", "request header 'Key: Value' (repeatable)")
	fs.IntVar(&cfg.concurrency, "c", 50, "concurrent connections (closed-loop)")
	fs.IntVar(&cfg.rps, "rps", 0, "target requests/sec (open-loop); 0 = closed-loop at -c")
	fs.DurationVar(&cfg.duration, "d", 30*time.Second, "test duration")
	fs.Int64Var(&cfg.total, "n", 0, "total requests (overrides -d when > 0)")
	fs.DurationVar(&cfg.timeout, "timeout", 30*time.Second, "per-request timeout")
	fs.BoolVar(&cfg.stream, "stream", false, "treat response as a stream: measure TTFB, drain to end")
	_ = fs.Parse(args)

	if cfg.url == "" {
		fmt.Fprintln(os.Stderr, "error: -url is required")
		fs.Usage()
		os.Exit(2)
	}
	if *bodyArg != "" {
		if strings.HasPrefix(*bodyArg, "@") {
			b, err := os.ReadFile((*bodyArg)[1:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading body file: %v\n", err)
				os.Exit(1)
			}
			cfg.body = b
		} else {
			cfg.body = []byte(*bodyArg)
		}
	}
	if cfg.method == "" {
		if cfg.body != nil {
			cfg.method = http.MethodPost
		} else {
			cfg.method = http.MethodGet
		}
	}

	runLoad(cfg)
}

func newClient(cfg loadConfig) *http.Client {
	// 给负载工具自身配足连接池，避免工具成为瓶颈（这正是被测 one-api 默认值偏小的点）。
	maxConns := max(cfg.concurrency, 100)
	tr := &http.Transport{
		MaxIdleConns:        maxConns * 2,
		MaxIdleConnsPerHost: maxConns,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   false, // 关掉 h2，逐连接更接近 keepalive 行为，便于观察连接复用
	}
	return &http.Client{Transport: tr, Timeout: cfg.timeout}
}

func runLoad(cfg loadConfig) {
	client := newClient(cfg)
	st := newStats()
	resultCh := make(chan result, 8192)

	var collectorWG sync.WaitGroup
	collectorWG.Add(1)
	go func() {
		defer collectorWG.Done()
		for r := range resultCh {
			st.add(r)
		}
	}()

	mode := "closed-loop"
	if cfg.rps > 0 {
		mode = fmt.Sprintf("open-loop %d rps", cfg.rps)
	}
	stop := "duration " + cfg.duration.String()
	if cfg.total > 0 {
		stop = fmt.Sprintf("%d requests", cfg.total)
	}
	fmt.Printf("load: %s %s | %s | %s | stream=%v\n",
		cfg.method, cfg.url, mode, stop, cfg.stream)

	start := time.Now()
	if cfg.rps > 0 {
		st.saturated = runOpenLoop(cfg, client, resultCh)
	} else {
		runClosedLoop(cfg, client, resultCh)
	}
	st.wall = time.Since(start)

	close(resultCh)
	collectorWG.Wait()
	st.report()
}

// runClosedLoop：固定 concurrency 个 worker 不断发请求，衡量在该并发下的极限吞吐。
func runClosedLoop(cfg loadConfig, client *http.Client, out chan<- result) {
	var sent int64
	deadline := time.Now().Add(cfg.duration)
	var wg sync.WaitGroup
	for i := 0; i < cfg.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if cfg.total > 0 {
					if atomic.AddInt64(&sent, 1) > cfg.total {
						return
					}
				} else if time.Now().After(deadline) {
					return
				}
				out <- doRequest(client, cfg)
			}
		}()
	}
	wg.Wait()
}

// runOpenLoop：按目标速率发请求，不受响应快慢影响（用于测固定 QPS 下的延迟）。
// in-flight 有上限，达到上限即记为 saturated —— 说明服务端跟不上目标速率。
func runOpenLoop(cfg loadConfig, client *http.Client, out chan<- result) int64 {
	interval := time.Second / time.Duration(cfg.rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	maxInFlight := max(cfg.rps*int(cfg.timeout.Seconds()+1), 1000)
	sem := make(chan struct{}, maxInFlight)
	deadline := time.Now().Add(cfg.duration)

	var saturated, sent int64
	var wg sync.WaitGroup
loop:
	for range ticker.C {
		if cfg.total > 0 {
			if sent >= cfg.total {
				break loop
			}
		} else if time.Now().After(deadline) {
			break loop
		}
		sent++
		select {
		case sem <- struct{}{}:
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				out <- doRequest(client, cfg)
			}()
		default:
			saturated++
		}
	}
	wg.Wait()
	return saturated
}

func doRequest(client *http.Client, cfg loadConfig) result {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	var bodyReader io.Reader
	if cfg.body != nil {
		bodyReader = strings.NewReader(string(cfg.body))
	}
	req, err := http.NewRequestWithContext(ctx, cfg.method, cfg.url, bodyReader)
	if err != nil {
		return result{err: err}
	}
	if cfg.body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, h := range cfg.headers {
		k, v, ok := strings.Cut(h, ":")
		if !ok {
			continue
		}
		req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return result{err: err, latency: time.Since(start)}
	}
	defer resp.Body.Close()

	var ttfb time.Duration
	var n int64
	if cfg.stream {
		// 读首字节记 TTFB，再把整个流读完。
		buf := make([]byte, 32*1024)
		first := true
		for {
			m, readErr := resp.Body.Read(buf)
			if m > 0 {
				if first {
					ttfb = time.Since(start)
					first = false
				}
				n += int64(m)
			}
			if readErr != nil {
				break
			}
		}
	} else {
		n, _ = io.Copy(io.Discard, resp.Body)
	}

	return result{
		latency: time.Since(start),
		ttfb:    ttfb,
		status:  resp.StatusCode,
		bytes:   n,
	}
}
