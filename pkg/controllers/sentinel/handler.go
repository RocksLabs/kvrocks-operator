package sentinel

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/k8s"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

type KVRocksSentinelHandler struct {
	instance *kvrocksv1alpha1.KVRocks
	key      types.NamespacedName
	k8s      *k8s.Client
	kvrocks  *kvrocks.Client
	log      logr.Logger
	pods     []string
	requeue  bool
}

func NewKVRocksSentinelHandler(
	k8s *k8s.Client,
	kvrocks *kvrocks.Client,
	logger logr.Logger,
	key types.NamespacedName,
	instance *kvrocksv1alpha1.KVRocks) *KVRocksSentinelHandler {
	return &KVRocksSentinelHandler{
		instance: instance,
		k8s:      k8s,
		kvrocks:  kvrocks,
		log:      logger,
		key:      key,
		requeue:  false,
	}
}

func (h *KVRocksSentinelHandler) Handle() (error, bool) {
	err := h.ensureKubernetes()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.ensureSentinel()
	if err != nil || h.requeue {
		return err, false
	}
	h.instance.Status.Status = kvrocksv1alpha1.StatusRunning
	if err := h.k8s.UpdateKVRocks(h.instance); err != nil {
		return err, false
	}
	return nil, true
}

func (h *KVRocksSentinelHandler) Requeue() bool {
	return h.requeue
}

func (h *KVRocksSentinelHandler) Finializer() error {
	return nil
}
