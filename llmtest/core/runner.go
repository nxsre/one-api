package core

import (
	"context"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// Result 单个用例的执行结果。
type Result struct {
	Target   string        `json:"target"`
	Scenario string        `json:"scenario"`
	Kind     EndpointKind  `json:"kind"`
	Status   string        `json:"status"` // PASS | FAIL | ERROR | SKIP
	Latency  time.Duration `json:"latency_ms"`
	Chunks   int           `json:"chunks"`
	Usage    Usage         `json:"usage"`
	Failures []string      `json:"failures,omitempty"`
	Err      string        `json:"error,omitempty"`
	Reason   string        `json:"reason,omitempty"` // SKIP 原因
	Raw      string        `json:"raw,omitempty"`    // 仅 verbose 时填充
}

const (
	StatusPass  = "PASS"
	StatusFail  = "FAIL"
	StatusError = "ERROR"
	StatusSkip  = "SKIP"
)

type job struct {
	target   Target
	provider Provider
	scenario Scenario
	cas      Case
}

func modelFor(m ModelSet, kind EndpointKind) string {
	switch kind {
	case KindCompletion:
		return m.Completion
	case KindEmbedding:
		return m.Embedding
	default:
		return m.Chat
	}
}

// FilterScenarios 依据 ID 列表筛选场景；ids 为空返回全部。
func FilterScenarios(all []Scenario, ids []string) []Scenario {
	if len(ids) == 0 {
		return all
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []Scenario
	for _, s := range all {
		if want[s.ID] {
			out = append(out, s)
		}
	}
	return out
}

// Run 执行配置中所有 target × 适用场景的测试。verbose 时在结果里附带原始响应。
func Run(cfg *Config, selected []string, verbose bool) []Result {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	client := NewClient(timeout)
	rng := rand.New(rand.NewSource(cfg.Seed))
	scenarios := FilterScenarios(AllScenarios(), selected)

	var jobs []job
	var results []Result // 先收集 SKIP/ERROR（构建期）

	// 构建期顺序执行，保证随机输入可复现（与并发无关）。
	for _, t := range cfg.Targets {
		prov, err := NewProvider(t.Protocol, client)
		if err != nil {
			results = append(results, Result{Target: t.Name, Status: StatusError, Err: err.Error()})
			continue
		}
		for _, s := range scenarios {
			if !prov.Supports(s.Kind) {
				results = append(results, Result{Target: t.Name, Scenario: s.ID, Kind: s.Kind, Status: StatusSkip, Reason: "协议不支持该接口形态"})
				continue
			}
			if modelFor(t.Models, s.Kind) == "" {
				results = append(results, Result{Target: t.Name, Scenario: s.ID, Kind: s.Kind, Status: StatusSkip, Reason: "未配置该形态的模型"})
				continue
			}
			cas := s.Build(rng, cfg.CustomInputs[s.ID], t.Models)
			jobs = append(jobs, job{target: t, provider: prov, scenario: s, cas: cas})
		}
	}

	// 执行期并发。
	resCh := make(chan Result, len(jobs))
	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resCh <- runJob(client, timeout, j, verbose)
		}(j)
	}
	wg.Wait()
	close(resCh)
	for r := range resCh {
		results = append(results, r)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Target != results[j].Target {
			return results[i].Target < results[j].Target
		}
		return results[i].Scenario < results[j].Scenario
	})
	return results
}

func runJob(_ *Client, timeout time.Duration, j job, verbose bool) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r := Result{Target: j.target.Name, Scenario: j.scenario.ID, Kind: j.cas.Kind}
	start := time.Now()

	if j.cas.Kind == KindEmbedding {
		resp, err := j.provider.Embedding(ctx, j.target, *j.cas.Emb)
		r.Latency = time.Since(start)
		if err != nil {
			r.Status = StatusError
			r.Err = err.Error()
			return r
		}
		r.Usage = resp.Usage
		r.Failures = Evaluate(j.cas.Exp, nil, resp)
		if verbose {
			r.Raw = string(resp.Raw)
		}
	} else {
		resp, err := j.provider.Chat(ctx, j.target, *j.cas.Chat)
		r.Latency = time.Since(start)
		if err != nil {
			r.Status = StatusError
			r.Err = err.Error()
			return r
		}
		r.Chunks = resp.Chunks
		r.Usage = resp.Usage
		r.Failures = Evaluate(j.cas.Exp, resp, nil)
		if verbose {
			r.Raw = string(resp.Raw)
		}
	}

	if len(r.Failures) == 0 {
		r.Status = StatusPass
	} else {
		r.Status = StatusFail
	}
	return r
}
