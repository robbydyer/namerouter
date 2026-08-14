package namerouter

import (
	"fmt"
	"net"
)

// httpsHost swaps any port in the request's Host header for the configured HTTPS port
func (n *NameRouter) httpsHost(host string) string {
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		// no port in the Host header
		hostname = host
	}

	if n.config.HTTPSPort == 0 || n.config.HTTPSPort == 443 {
		return hostname
	}

	return net.JoinHostPort(hostname, fmt.Sprintf("%d", n.config.HTTPSPort))
}

func getExternalHosts(nh []*Namehost) []string {
	hosts := []string{}

	for _, n := range nh {
		hosts = append(hosts, n.ExternalHosts...)
	}

	return hosts
}
