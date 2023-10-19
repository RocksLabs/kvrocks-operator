package cluster

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	sentinel "github.com/RocksLabs/kvrocks-operator/pkg/controllers/sentinel"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksClusterHandler) ensureSentinel() error {
	if h.instance.Status.Shrink != nil {
		return nil
	}
	// add Finalizer
	if !controllerutil.ContainsFinalizer(h.instance, kvrocksv1alpha1.KVRocksFinalizer) {
		controllerutil.AddFinalizer(h.instance, kvrocksv1alpha1.KVRocksFinalizer)
		if err := h.k8s.UpdateKVRocks(h.instance); err != nil {
			return err
		}
	}
	// notify sentinel to update
	if v, ok := h.instance.Labels[resources.MonitoredBy]; ok {
		return sentinel.UpdateSentinelAnnotationCount(h.k8s, h.instance.Namespace, v)
	}

	return nil
}
