package namerouter

func getHosts(nh []*Namehost) []string {
	hosts := []string{}

	for _, n := range nh {
		for _, host := range n.Hosts {
			if host != "default" {
				hosts = append(hosts, host)
			}
		}
	}

	return hosts
}
