package namerouter

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func (n *NameRouter) getVisitor(ip string) *rate.Limiter {
	n.Lock()
	defer n.Unlock()

	v, ok := n.visitors[ip]
	if !ok || v == nil {
		l := rate.NewLimiter(50, 100)
		n.visitors[ip] = &visitor{
			limiter:  l,
			lastSeen: time.Now(),
		}
		return l
	}

	v.lastSeen = time.Now()

	return v.limiter
}

func (n *NameRouter) visitorCleanup(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}

		n.Lock()
		for ip, v := range n.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(n.visitors, ip)
			}
		}
		n.Unlock()
	}
}
