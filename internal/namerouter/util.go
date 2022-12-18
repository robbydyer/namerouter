package namerouter

func getHosts(nh []*Namehost) []string {
	hosts := []string{}

	for _, n := range nh {
		hosts = append(hosts, n.Hosts...)
	}

	return hosts
}
