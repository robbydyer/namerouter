package namerouter

func getExternalHosts(nh []*Namehost) []string {
	hosts := []string{}

	for _, n := range nh {
		hosts = append(hosts, n.ExternalHosts...)
	}

	return hosts
}
