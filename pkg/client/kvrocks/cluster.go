package kvrocks

import (
	"strings"
)

func (s *Client) ClusterNodeInfo(ip, password string) (*Node, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	info, err := c.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, err
	}
	return parseNodeInfo(info)
}

func parseNodeInfo(info string) (*Node, error) {
	node := &Node{}
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		fields := strings.Split(line, " ")
		if len(fields) < 8 {
			// last line is always empty
			continue
		}
		id := fields[0]
		flags := fields[2]
		if strings.Contains(flags, "myself") {
			node.NodeId = id
			node.IP = strings.Split(fields[1], ":")[0]
			if strings.Contains(flags, RoleMaster) {
				node.Role = RoleMaster
				node.Slots = SlotsToInt(fields[8:])
			} else if strings.Contains(flags, RoleSlaver) {
				node.Role = RoleSlaver
				node.Master = fields[3]
			}
		}
	}
	return node, nil
}
