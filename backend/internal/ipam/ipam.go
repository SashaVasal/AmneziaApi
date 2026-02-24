package ipam

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Allocator struct {
	baseSubnet string
	startHost  int
	endHost    int
}

func NewAllocator(baseSubnet string) *Allocator {
	return &Allocator{
		baseSubnet: baseSubnet,
		startHost:  2,
		endHost:    254,
	}
}

func (a *Allocator) NextFreeIP(allocated []string) (string, error) {
	used := make(map[int]struct{}, len(allocated))

	for _, value := range allocated {
		ipOnly := strings.TrimSpace(value)
		if strings.Contains(ipOnly, "/") {
			parsed, _, err := net.ParseCIDR(ipOnly)
			if err != nil {
				continue
			}
			ipOnly = parsed.String()
		}

		parts := strings.Split(ipOnly, ".")
		if len(parts) != 4 {
			continue
		}
		host, err := strconv.Atoi(parts[3])
		if err != nil {
			continue
		}
		used[host] = struct{}{}
	}

	for host := a.startHost; host <= a.endHost; host++ {
		if _, ok := used[host]; ok {
			continue
		}
		return fmt.Sprintf("10.8.0.%d/32", host), nil
	}

	return "", fmt.Errorf("no free addresses in %s", a.baseSubnet)
}
