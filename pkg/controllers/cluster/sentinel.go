package cluster

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/controllers/common"
)

func (h *KVRocksClusterHandler) ensureSentinel() error {
	if h.instance.Status.Shrink != nil {
		return nil
	}
	commHandler := common.NewCommandHandler(h.instance, h.k8s, h.kvrocks, h.password)
	var err error
	for index, sts := range h.stsNodes {
		for _, node := range sts {
			if node.Role == kvrocks.RoleMaster {
				h.requeue, err = commHandler.EnsureSentinel(node.IP, index)
				if err != nil {
					return err
				}
			}
		}
	}
	if !controllerutil.ContainsFinalizer(h.instance, kvrocksv1alpha1.KVRocksFinalizer) {
		controllerutil.AddFinalizer(h.instance, kvrocksv1alpha1.KVRocksFinalizer)
		if err = h.k8s.UpdateKVRocks(h.instance); err != nil {
			return err
		}
	}
	h.log.Info("sentinel monitor ready")
	return nil
}
