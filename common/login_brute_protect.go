package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
	"sync"
	"time"
)

const loginBruteKeyPrefix = "login_brute:"
const loginBruteCompositeSep = "\x00"

var (
	LoginBruteIPFailMax    = 20
	LoginBrutePairFailMax  = 5
	LoginBruteFailWindow   = time.Minute
	LoginBruteLockDuration = time.Minute
)

func normalizeLoginIdentifier(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func loginBruteIPShortKey(clientIP string) string {
	return "i:" + strings.TrimSpace(clientIP)
}

func loginBruteCompositeShortKey(clientIP, username string) string {
	userNorm := normalizeLoginIdentifier(username)
	if userNorm == "" {
		return ""
	}
	ipNorm := strings.TrimSpace(clientIP)
	raw := ipNorm + loginBruteCompositeSep + userNorm
	s := sha256.Sum256([]byte(raw))
	return "p:" + hex.EncodeToString(s[:])
}

func ResolveClientIPForLoginBrute(xForwardedFor, ginClientIP string) string {
	if !LoginBruteTrustXForwardedFor {
		return ginClientIP
	}
	ff := strings.TrimSpace(xForwardedFor)
	if ff == "" {
		return ginClientIP
	}
	first := strings.TrimSpace(strings.Split(ff, ",")[0])
	if host, _, err := net.SplitHostPort(first); err == nil {
		first = host
	}
	first = strings.TrimSpace(first)
	if strings.HasPrefix(first, "\"") && strings.HasSuffix(first, "\"") && len(first) >= 2 {
		first = strings.Trim(first, "\"")
	}
	if ip := net.ParseIP(first); ip != nil {
		return ip.String()
	}
	return ginClientIP
}

func IsLoginBruteLocked(clientIP, username string) bool {
	ipKey := loginBruteIPShortKey(clientIP)
	if !RedisEnabled || RDB == nil {
		if isLoginBruteLockedMemory(ipKey) {
			return true
		}
		short := loginBruteCompositeShortKey(clientIP, username)
		if short == "" {
			return false
		}
		return isLoginBruteLockedMemory(short)
	}
	ctx := context.Background()
	keys := []string{loginBruteKeyPrefix + "b:" + ipKey}
	if short := loginBruteCompositeShortKey(clientIP, username); short != "" {
		keys = append(keys, loginBruteKeyPrefix+"b:"+short)
	}
	n, err := RDB.Exists(ctx, keys...).Result()
	if err != nil {
		return false
	}
	return n > 0
}

func RecordLoginBruteFailure(clientIP, username string) {
	ipKey := loginBruteIPShortKey(clientIP)
	if !RedisEnabled || RDB == nil {
		recordLoginBruteFailureMemory(ipKey, LoginBruteIPFailMax)
		if short := loginBruteCompositeShortKey(clientIP, username); short != "" {
			recordLoginBruteFailureMemory(short, LoginBrutePairFailMax)
		}
		return
	}
	ctx := context.Background()
	recordOneRedis(ctx, ipKey, LoginBruteIPFailMax)
	if short := loginBruteCompositeShortKey(clientIP, username); short != "" {
		recordOneRedis(ctx, short, LoginBrutePairFailMax)
	}
}

func recordOneRedis(ctx context.Context, shortKey string, maxFailures int) {
	ckey := loginBruteKeyPrefix + "c:" + shortKey
	n, err := RDB.Incr(ctx, ckey).Result()
	if err != nil {
		return
	}
	if n == 1 {
		_ = RDB.Expire(ctx, ckey, LoginBruteFailWindow).Err()
	}
	if int(n) >= maxFailures {
		bkey := loginBruteKeyPrefix + "b:" + shortKey
		_ = RDB.Set(ctx, bkey, "1", LoginBruteLockDuration).Err()
		_ = RDB.Del(ctx, ckey).Err()
	}
}

func ClearLoginBruteState(clientIP, username string) {
	ipKey := loginBruteIPShortKey(clientIP)
	if !RedisEnabled || RDB == nil {
		clearLoginBrutePairMemory(clientIP, username)
		return
	}
	ctx := context.Background()
	keys := []string{
		loginBruteKeyPrefix + "c:" + ipKey,
		loginBruteKeyPrefix + "b:" + ipKey,
	}
	if short := loginBruteCompositeShortKey(clientIP, username); short != "" {
		keys = append(keys,
			loginBruteKeyPrefix+"c:"+short,
			loginBruteKeyPrefix+"b:"+short,
		)
	}
	_ = RDB.Del(ctx, keys...).Err()
}

type bruteMemEntry struct {
	mu          sync.Mutex
	failures    int
	windowStart time.Time
	lockedUntil time.Time
}

var bruteMemStore sync.Map

func memEntryFor(key string) *bruteMemEntry {
	v, _ := bruteMemStore.LoadOrStore(key, &bruteMemEntry{})
	return v.(*bruteMemEntry)
}

func isLoginBruteLockedMemory(shortKey string) bool {
	now := time.Now()
	if e := findMemEntry(shortKey); e != nil {
		e.mu.Lock()
		locked := now.Before(e.lockedUntil)
		e.mu.Unlock()
		return locked
	}
	return false
}

func findMemEntry(key string) *bruteMemEntry {
	v, ok := bruteMemStore.Load(key)
	if !ok {
		return nil
	}
	return v.(*bruteMemEntry)
}

func recordLoginBruteFailureMemory(shortKey string, maxFailures int) {
	bumpMem(shortKey, maxFailures)
}

func bumpMem(key string, maxFailures int) {
	now := time.Now()
	e := memEntryFor(key)
	e.mu.Lock()
	defer e.mu.Unlock()
	if now.Before(e.lockedUntil) {
		return
	}
	if e.failures == 0 || now.Sub(e.windowStart) > LoginBruteFailWindow {
		e.failures = 1
		e.windowStart = now
		return
	}
	e.failures++
	if e.failures >= maxFailures {
		e.lockedUntil = now.Add(LoginBruteLockDuration)
		e.failures = 0
		e.windowStart = time.Time{}
	}
}

func clearLoginBrutePairMemory(clientIP, username string) {
	bruteMemStore.Delete(loginBruteIPShortKey(clientIP))
	if short := loginBruteCompositeShortKey(clientIP, username); short != "" {
		bruteMemStore.Delete(short)
	}
}
