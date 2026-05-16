package routing

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"

	dbmodel "github.com/songquanpeng/one-api/model"
)

// SelectOpts 单次渠道挑选参数。
type SelectOpts struct {
	StickyKey           string
	RequestModel        string
	ExcludeChannelIDs   map[int]struct{}
	SkipCircuitDisabled bool
}

// PickChannel 在已按优先级降序排好的渠道列表中选择一台（尊重优先级分层与 ignoreFirstPriority 语义）。
func PickChannel(channels []*dbmodel.Channel, ignoreFirstPriority bool, opts SelectOpts) (*dbmodel.Channel, error) {
	cand, err := sliceCandidatesForPriority(channels, ignoreFirstPriority)
	if err != nil {
		return nil, err
	}
	cand = filterCandidates(cand, opts)
	if len(cand) == 0 {
		return nil, errors.New("channel not found")
	}
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].Id < cand[j].Id })

	pol := CurrentRoutingPolicy()
	cand = ApplyDirectionLayer(cand, pol, opts)
	if len(cand) == 0 {
		return nil, errors.New("channel not found")
	}
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].Id < cand[j].Id })

	if pol.SelectionMode == SelectionConsistentHash && opts.StickyKey != "" {
		return pickConsistentHash(cand, opts.StickyKey, pol.ConsistentHashSeed, opts.RequestModel), nil
	}
	return pickWeightedRandom(cand), nil
}

func sliceCandidatesForPriority(channels []*dbmodel.Channel, ignoreFirstPriority bool) ([]*dbmodel.Channel, error) {
	if len(channels) == 0 {
		return nil, errors.New("channel not found")
	}
	firstPri := channels[0].GetPriority()
	firstPriEnd := len(channels)
	for i := 1; i < len(channels); i++ {
		if channels[i].GetPriority() != firstPri {
			firstPriEnd = i
			break
		}
	}
	var cand []*dbmodel.Channel
	if !ignoreFirstPriority {
		cand = channels[:firstPriEnd]
	} else if firstPriEnd < len(channels) {
		cand = channels[firstPriEnd:]
	} else {
		cand = channels[:firstPriEnd]
	}
	return cand, nil
}

func filterCandidates(in []*dbmodel.Channel, opts SelectOpts) []*dbmodel.Channel {
	out := make([]*dbmodel.Channel, 0, len(in))
	for _, ch := range in {
		if opts.ExcludeChannelIDs != nil {
			if _, ok := opts.ExcludeChannelIDs[ch.Id]; ok {
				continue
			}
		}
		if opts.SkipCircuitDisabled && IsCircuitOpen(ch.Id) {
			continue
		}
		out = append(out, ch)
	}
	return out
}

func effectiveWeight(ch *dbmodel.Channel) uint {
	w := ch.GetWeight()
	mul := GetManualWeightMultiplier(ch.Id)
	p := CurrentRoutingPolicy()
	if p.AutoAdaptiveEnabled && ch.RoutingAdaptiveEffective() {
		mul *= GetAutoWeightMultiplier(ch.Id)
	}
	v := float64(w) * mul
	if v < 1 {
		return 1
	}
	return uint(v + 0.999) // ceil without import math
}

func pickWeightedRandom(cand []*dbmodel.Channel) *dbmodel.Channel {
	var sum uint64
	weights := make([]uint, len(cand))
	for i, ch := range cand {
		w := effectiveWeight(ch)
		weights[i] = w
		sum += uint64(w)
	}
	if sum == 0 {
		return cand[rand.Intn(len(cand))]
	}
	r := rand.Uint64() % sum
	var acc uint64
	for i, w := range weights {
		acc += uint64(w)
		if r < acc {
			return cand[i]
		}
	}
	return cand[len(cand)-1]
}

func pickConsistentHash(cand []*dbmodel.Channel, stickyKey, seed, requestModel string) *dbmodel.Channel {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%s|%s|%s", seed, stickyKey, requestModel)
	sumHash := h.Sum64()
	var sum uint64
	weights := make([]uint, len(cand))
	for i, ch := range cand {
		w := effectiveWeight(ch)
		weights[i] = w
		sum += uint64(w)
	}
	if sum == 0 {
		return cand[int(sumHash)%len(cand)]
	}
	r := sumHash % sum
	var acc uint64
	for i, w := range weights {
		acc += uint64(w)
		if r < acc {
			return cand[i]
		}
	}
	return cand[len(cand)-1]
}
