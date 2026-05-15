package localnet

import "net"

func LANAddresses() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip == nil {
				continue
			}
			ip = ip.To4()
			if ip == nil || !isPrivateIPv4(ip) {
				continue
			}
			out = append(out, ip.String())
		}
	}
	return out
}

func isPrivateIPv4(ip net.IP) bool {
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}
