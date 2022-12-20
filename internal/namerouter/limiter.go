package namerouter

import (
	"context"
	"net"
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
		ipAddr := net.ParseIP(ip)
		var l *rate.Limiter
		if ipAddr.IsPrivate() {
			l = rate.NewLimiter(rate.Limit(n.config.RateLimits.Internal.Rate), n.config.RateLimits.Internal.Burst)
		} else {
			l = rate.NewLimiter(rate.Limit(n.config.RateLimits.External.Rate), n.config.RateLimits.External.Burst)
		}
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
