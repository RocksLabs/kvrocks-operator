package common

import (
	"fmt"
	"strings"

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

func (h *CommandHandler) EnsureTopo() (bool, error) {
	if h.instance.Status.Rebalance {
		return false, nil
	}
	topoMsg := ""
	for _, sts := range h.instance.Status.Topo {
		for _, node := range sts.Topology {
			if node.Failover {
				continue
			}
			topoMsg += getTopoMsg(node)
		}
	}
ensureTopo:
	for _, sts := range h.instance.Status.Topo {
		for _, node := range sts.Topology {
			if node.Failover {
				continue
			}
			if err := h.kvrocks.SetTopoMsg(node.Ip, h.password, topoMsg, h.instance.Status.Version); err != nil {
				if err.Error() == kvrocks.ClusterInvalidVersion {
					h.instance.Status.Version++
					goto ensureTopo
				}
				return false, err
			}
		}
	}
	if h.instance.Status.Status != kvrocksv1alpha1.StatusRunning {
		h.instance.Status.Status = kvrocksv1alpha1.StatusRunning
	}
	if err := h.k8s.UpdateKVRocks(h.instance); err != nil {
		return true, err
	}
	return false, nil
}

func getTopoMsg(node kvrocksv1alpha1.KVRocksTopology) string {
	var msg string
	if node.Role == kvrocks.RoleMaster {
		msg = fmt.Sprintf("%s %s %d master - %v", node.NodeId, node.Ip, kvrocks.KVRocksPort, node.Slots)
		msg = strings.ReplaceAll(msg, "[", "")
		msg = strings.ReplaceAll(msg, "]", "")
	} else {
		msg = fmt.Sprintf("%s %s %d slave %s", node.NodeId, node.Ip, kvrocks.KVRocksPort, node.MasterId)
	}
	return msg + "\n"
}
