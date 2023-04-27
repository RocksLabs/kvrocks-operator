package common

import (
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *CommandHandler) ResizeStatefulSet(stsNodes []*kvrocks.Node, index ...int) (bool, error) {
	delta := len(stsNodes) - int(h.instance.Spec.Replicas)
	if delta == 0 {
		return false, nil
	}
	// scaling down,delete slave node
	if delta > 0 {
		sts := resources.NewReplicationStatefulSet(h.instance)
		if len(index) > 0 {
			sts = resources.NewClusterStatefulSet(h.instance, index[0])
		}
		reserve := make([]int, 0)
		masterID := 0
		for i := len(stsNodes) - 1; i >= 0; i-- {
			if delta > 0 {
				if stsNodes[i].Role != kvrocks.RoleMaster {
					reserve = append(reserve, stsNodes[i].PodIndex)
					delta--
				} else {
					masterID = stsNodes[i].PodIndex
				}
			}
			if delta == 0 {
				break
			}
		}
		for i := len(reserve) - 1; i >= 0; i-- {
			id := reserve[i]
			stsNodes[id] = nil
			// no need to reserve ordinal larger than masterID
			if id > masterID {
				continue
			}
			sts.Spec.ReserveOrdinals = append(sts.Spec.ReserveOrdinals, id)
		}
		if err := h.k8s.CreateOrUpdateStatefulSet(sts); err != nil {
			return true, err
		}
	}
	return true, nil
}
