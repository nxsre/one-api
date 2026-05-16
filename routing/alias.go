package routing

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// ResolveModelAliases 应用链式映射与多候选别名（无分组覆盖）。
func ResolveModelAliases(logical string) (string, error) {
	return ResolveModelAliasesForGroup("", logical)
}

// EffectiveAliasPolicyForGroup 合并全局别名与分组覆盖。
func EffectiveAliasPolicyForGroup(group string) ModelAliasPolicy {
	p := CurrentModelAliasPolicy()
	g := strings.TrimSpace(group)
	if g == "" {
		return p
	}
	frag, ok := p.GroupOverrides[g]
	if !ok {
		return p
	}
	out := ModelAliasPolicy{
		MaxSteps:       p.MaxSteps,
		Defaults:       nil,
		GroupOverrides: nil,
	}
	out.Chains = map[string]string{}
	for k, v := range p.Chains {
		out.Chains[k] = v
	}
	for k, v := range frag.Chains {
		out.Chains[k] = v
	}
	out.Aliases = map[string][]AliasTarget{}
	for k, v := range p.Aliases {
		cp := make([]AliasTarget, len(v))
		copy(cp, v)
		out.Aliases[k] = cp
	}
	for k, v := range frag.Aliases {
		out.Aliases[k] = append([]AliasTarget(nil), v...)
	}
	return out
}

// ResolveModelAliasesForGroup 分组感知别名解析。
func ResolveModelAliasesForGroup(group, logical string) (string, error) {
	logical = strings.TrimSpace(logical)
	if logical == "" {
		return "", errors.New("empty model")
	}
	pol := EffectiveAliasPolicyForGroup(group)
	cur, err := applyChains(logical, pol.Chains, pol.MaxSteps)
	if err != nil {
		return "", err
	}
	tgts, ok := pol.Aliases[cur]
	if !ok || len(tgts) == 0 {
		return cur, nil
	}
	return pickAliasTarget(tgts), nil
}

// ValidateModelAliasPolicy 校验链无环（含分组覆盖合并）。
func ValidateModelAliasPolicy(p ModelAliasPolicy) error {
	maxSteps := p.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 32
	}
	if p.Chains == nil {
		p.Chains = map[string]string{}
	}
	for start := range p.Chains {
		if _, err := applyChains(start, p.Chains, maxSteps); err != nil {
			return fmt.Errorf("global chain from %q: %w", start, err)
		}
	}
	for g, frag := range p.GroupOverrides {
		ch := map[string]string{}
		for k, v := range p.Chains {
			ch[k] = v
		}
		for k, v := range frag.Chains {
			ch[k] = v
		}
		ms := p.MaxSteps
		if ms <= 0 {
			ms = 32
		}
		for start := range ch {
			if _, err := applyChains(start, ch, ms); err != nil {
				return fmt.Errorf("group %q chain from %q: %w", g, start, err)
			}
		}
	}
	return nil
}

func applyChains(start string, chains map[string]string, maxSteps int) (string, error) {
	cur := strings.TrimSpace(start)
	visited := map[string]struct{}{cur: {}}
	for step := 0; step < maxSteps; step++ {
		next, ok := chains[cur]
		if !ok {
			break
		}
		next = strings.TrimSpace(next)
		if next == "" {
			break
		}
		if _, dup := visited[next]; dup {
			return "", fmt.Errorf("model alias chain cycle detected at %q -> %q", cur, next)
		}
		visited[next] = struct{}{}
		cur = next
	}
	return cur, nil
}

func pickAliasTarget(targets []AliasTarget) string {
	type item struct {
		model    string
		priority int64
		weight   int
	}
	items := make([]item, 0, len(targets))
	for _, t := range targets {
		w := t.Weight
		if w <= 0 {
			w = 1
		}
		items = append(items, item{model: strings.TrimSpace(t.Model), priority: t.Priority, weight: w})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority > items[j].priority
		}
		return items[i].model < items[j].model
	})
	topP := items[0].priority
	layer := make([]item, 0)
	for _, it := range items {
		if it.priority == topP {
			layer = append(layer, it)
		}
	}
	var sum int
	for _, it := range layer {
		sum += it.weight
	}
	r := rand.Intn(sum)
	acc := 0
	for _, it := range layer {
		acc += it.weight
		if r < acc {
			return it.model
		}
	}
	return layer[len(layer)-1].model
}

// ValidateModelAliasPolicyRaw 解析 JSON 后校验。
func ValidateModelAliasPolicyRaw(raw string) error {
	var p ModelAliasPolicy
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return err
	}
	if p.Chains == nil {
		p.Chains = map[string]string{}
	}
	if p.MaxSteps <= 0 {
		p.MaxSteps = 32
	}
	return ValidateModelAliasPolicy(p)
}
