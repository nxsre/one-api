package routing

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	dbmodel "github.com/songquanpeng/one-api/model"

	"github.com/songquanpeng/one-api/common"
)

// ApplyDirectionLayer 在同优先级候选内按 Provider（Direction）筛选后再选 Route。
func ApplyDirectionLayer(cand []*dbmodel.Channel, pol RoutingPolicy, opts SelectOpts) []*dbmodel.Channel {
	if !pol.DirectionEnabled || len(cand) <= 1 {
		return cand
	}
	provs := map[string][]*dbmodel.Channel{}
	for _, ch := range cand {
		p := ch.RoutingProviderTag()
		provs[p] = append(provs[p], ch)
	}
	if len(provs) <= 1 {
		return cand
	}
	picked := pickDirectionProviderKey(provs, pol, opts.RequestModel)
	return provs[picked]
}

type scoredProv struct {
	name string
	s    float64
}

func pickDirectionProviderKey(provs map[string][]*dbmodel.Channel, pol RoutingPolicy, model string) string {
	day := time.Now().UTC().Format("20060102")
	list := make([]scoredProv, 0, len(provs))
	for name := range provs {
		list = append(list, scoredProv{name: name, s: providerScore(day, name, model, pol)})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].s == list[j].s {
			return list[i].name < list[j].name
		}
		return list[i].s < list[j].s
	})
	best := list[0].name

	probe := pol.DirectionProbeRatio
	if probe < pol.DirectionMinProbe {
		probe = pol.DirectionMinProbe
	}
	if probe > 1 {
		probe = 1
	}
	if rand.Float64() >= probe {
		return best
	}
	if len(list) == 1 {
		return best
	}
	switch pol.ProbePickStrategy {
	case "worst_first":
		return list[len(list)-1].name
	default:
		if len(list) <= 2 {
			return list[rand.Intn(len(list))].name
		}
		idx := 1 + rand.Intn(len(list)-1)
		return list[idx].name
	}
}

func providerScore(day, provider, model string, pol RoutingPolicy) float64 {
	ok, fail, latSum, latN := fetchProviderDayAggregate(day, provider, model)
	total := ok + fail
	errRatio := 0.0
	if total > 0 {
		errRatio = float64(fail) / float64(total)
	}
	latAvg := 800.0
	if latN > 0 {
		latAvg = float64(latSum) / float64(latN)
	}
	latNorm := latAvg / 8000.0
	if latNorm > 1 {
		latNorm = 1
	}
	return pol.DirectionLatencyW*latNorm + pol.DirectionErrorW*errRatio
}

func fetchProviderDayAggregate(day, provider, model string) (ok, fail, latSum, latN int64) {
	if !common.RedisEnabled || common.RDB == nil {
		return 0, 0, 0, 0
	}
	ctx := context.Background()
	key := fmt.Sprintf("oa:rt:d:%s:%s:%s", day, provider, trimModel(model))
	h, err := common.RDB.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, 0, 0, 0
	}
	parse := func(s string) int64 {
		v, _ := strconv.ParseInt(s, 10, 64)
		return v
	}
	return parse(h["ok"]), parse(h["fail"]), parse(h["lat_sum_ms"]), parse(h["lat_n"])
}
