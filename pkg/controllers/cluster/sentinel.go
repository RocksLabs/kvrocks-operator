package cluster

import (
	"strconv"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
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
		return h.updateSentinelAnnotationCount(v)
	}

	return nil
}

func (h *KVRocksClusterHandler) updateSentinelAnnotationCount(sentinelName string) error {
	sentinel, err := h.k8s.GetKVRocks(types.NamespacedName{
		Namespace: h.instance.Namespace,
		Name:      sentinelName,
	})
	if err != nil {
		return err
	}
	annotations := sentinel.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	count, ok := annotations["change-count"]
	if !ok {
		count = "0"
	}
	countInt, err := strconv.Atoi(count)
	if err != nil {
		return err
	}
	countInt++
	annotations["change-count"] = strconv.Itoa(countInt)
	sentinel.SetAnnotations(annotations)
	if err := h.k8s.UpdateKVRocks(sentinel); err != nil {
		return err
	}
	h.log.V(1).Info("sentinel monitor ready")
	return nil
}
