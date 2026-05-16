/**
 * 与后端 routing/policy.go 中 Option JSON 字段对齐的示例（合法 JSON，可按需删减）。
 */
export const ROUTING_POLICY_JSON_SAMPLES = {
  RoutingPolicy: `{
  "selection_mode": "weighted_random",
  "consistent_hash_seed": "",
  "sticky_source": "token_id",
  "direction_enabled": true,
  "direction_probe_ratio": 0.1,
  "direction_min_probe_ratio": 0.02,
  "direction_latency_weight": 1,
  "direction_error_weight": 2,
  "probe_pick_strategy": "secondary_uniform",
  "auto_adaptive_enabled": true,
  "adaptive_interval_sec": 60,
  "circuit_fail_threshold": 8,
  "circuit_cooldown_sec": 60,
  "half_open_probe_ms": 8000,
  "adaptive_aggressive_penalty": 0.92,
  "adaptive_gentle_boost": 1.01,
  "adaptive_err_ratio_threshold": 0.25,
  "auto_weight_min": 0.05,
  "auto_weight_max": 3,
  "aggregation_interval_sec": 60
}`,

  RelayRetryPolicy: `{
  "max_retries_override": 3,
  "base_backoff_ms": 150,
  "max_backoff_ms": 4000,
  "retry_http_status_allowlist": [408, 409, 429, 500, 502, 503],
  "retry_http_status_denylist": [],
  "force_different_channel_each_attempt": true
}`,

  ModelAliasPolicy: `{
  "max_chain_steps": 32,
  "chains": {
    "gpt-public": "gpt-4o-mini",
    "route-heavy": "gpt-4o"
  },
  "aliases": {
    "gpt-4o-mini": [
      { "model": "upstream-mini-primary", "weight": 10, "priority": 100 },
      { "model": "upstream-mini-backup", "weight": 5, "priority": 90 }
    ],
    "gpt-4o": [
      { "model": "upstream-4o-azure", "weight": 8, "priority": 100 },
      { "model": "upstream-4o-openai", "weight": 6, "priority": 80 }
    ]
  },
  "defaults": {},
  "group_overrides": {
    "vip": {
      "chains": {
        "gpt-public": "gpt-4o"
      },
      "aliases": {
        "gpt-4o": [
          { "model": "vip-only-channel-model", "weight": 1, "priority": 0 }
        ]
      }
    }
  }
}`,

  ModelRateLimitPolicy: `{
  "enabled": true,
  "by_token": {
    "*": {
      "qps": 80,
      "burst": 120,
      "concurrency": 20,
      "daily_quota": 0
    },
    "gpt-4o": {
      "qps": 6,
      "burst": 12,
      "concurrency": 4,
      "daily_quota": 0
    }
  },
  "by_user": {},
  "by_group": {}
}`
};
