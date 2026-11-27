package httpapi

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimit cấu hình token bucket: Burst yêu cầu liên tiếp, hồi 1 lượt mỗi Every.
type RateLimit struct {
	Burst int
	Every time.Duration
}

var errRateLimited = &httpError{
	status: http.StatusTooManyRequests,
	code:   "RATE_LIMITED",
	msg:    "Bạn thao tác quá nhanh, vui lòng thử lại sau ít phút",
}

// maxBuckets giới hạn bộ nhớ: vượt ngưỡng thì dọn các bucket đã đầy lại.
const maxBuckets = 10_000

type bucket struct {
	tokens float64
	last   time.Time
}

// rateLimiter giới hạn theo IP trong bộ nhớ (một instance API — đủ cho phạm vi đồ án).
type rateLimiter struct {
	mu      sync.Mutex
	cfg     RateLimit
	now     func() time.Time
	buckets map[string]*bucket
}

func newRateLimiter(cfg RateLimit, now func() time.Time) *rateLimiter {
	return &rateLimiter{cfg: cfg, now: now, buckets: make(map[string]*bucket)}
}

// allow trả true nếu còn lượt; nếu hết, trả thời gian phải chờ.
func (l *rateLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	burst := float64(l.cfg.Burst)
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= maxBuckets {
			l.evictFull(now)
		}
		b = &bucket{tokens: burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(burst, b.tokens+l.refill(now.Sub(b.last)))
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) * float64(l.cfg.Every))
	return false, wait
}

func (l *rateLimiter) refill(elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(elapsed) / float64(l.cfg.Every)
}

func (l *rateLimiter) evictFull(now time.Time) {
	for k, b := range l.buckets {
		if b.tokens+l.refill(now.Sub(b.last)) >= float64(l.cfg.Burst) {
			delete(l.buckets, k)
		}
	}
}

// middleware áp giới hạn theo IP của kết nối (RemoteAddr). Không tin X-Forwarded-For vì giả mạo được.
func (l *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := l.allow(clientIP(r))
		if !ok {
			secs := int(math.Ceil(wait.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(max(secs, 1)))
			writeError(w, r, errRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
