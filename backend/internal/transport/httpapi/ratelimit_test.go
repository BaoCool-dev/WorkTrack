package httpapi

import (
	"testing"
	"time"
)

func TestRateLimiter_RefillsOverTime_CF01(t *testing.T) {
	now := testNow
	l := newRateLimiter(RateLimit{Burst: 2, Every: 10 * time.Second}, func() time.Time { return now })

	for i := range 2 {
		if ok, _ := l.allow("1.2.3.4"); !ok {
			t.Fatalf("lần %d bị chặn", i+1)
		}
	}
	ok, wait := l.allow("1.2.3.4")
	if ok || wait != 10*time.Second {
		t.Fatalf("ok=%v wait=%v, want chặn 10s", ok, wait)
	}
	if ok, _ := l.allow("5.6.7.8"); !ok {
		t.Error("IP khác bị chặn chung bucket")
	}
	now = now.Add(10 * time.Second)
	if ok, _ := l.allow("1.2.3.4"); !ok {
		t.Error("chưa hồi lượt sau 10s")
	}
}
