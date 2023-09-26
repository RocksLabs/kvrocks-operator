package kvrocks

import (
	"strings"
)

const (
	ClusterNotInitErr     = "CLUSTERDOWN The cluster is not initialized"
	ClusterAlreadyMigrate = "Can't migrate slot which has been migrated"
	ClusterSlotInvalid    = "Can't migrate slot which doesn't belong to me"
	ClusterVersionInvalid = "Invalid cluster version"
	ClusterInvalidVersion = "Invalid version of cluster"
)

// SetClusterID sets the cluster nodeID
func (s *client) SetClusterID(ip, password, nodeID string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "SETNODEID", nodeID).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("set cluster nodeID successfully", "ip", ip, "nodeId", nodeID)
	return nil
}

// SetTopoMsg sets the cluster topoMsg
func (s *client) SetTopoMsg(ip, password, topoMsg string, version int) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "SETNODES", topoMsg, version).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("clusterx setnodes successfully", "ip", ip)
	return nil
}

// MoveSlots moves the slots to the dstNodeId
func (s *client) MoveSlots(ip, password string, slot int, dstNodeId string) bool {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "MIGRATE", slot, dstNodeId).Err(); err != nil && (err.Error() == ClusterAlreadyMigrate || err.Error() == ClusterSlotInvalid) {
		return true
	}
	return false
}

// ResetSlot resets the slot to the dstNodeId
func (s *client) ResetSlot(ip, password string, slot, version int, dstNodeId string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.Do(ctx, "CLUSTERX", "SETSLOT", slot, "NODE", dstNodeId, version).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("clusterx setslot successfully", "ip", ip, "node", dstNodeId, "slot", slot, "version", version)
	return nil
}

// ClusterVersion returns the cluster version
func (s *client) ClusterVersion(ip, password string) (int, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	result, err := c.Do(ctx, "CLUSTERX", "VERSION").Int()
	if err != nil {
		return -1, err
	}
	return result, nil
}

// ClusterNodeInfo returns the cluster node info
func (s *client) ClusterNodeInfo(ip, password string) (*Node, error) {
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
