package standard

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/k8s"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/controllers/common"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

type KVRocksStandardHandler struct {
	instance *kvrocksv1alpha1.KVRocks
	k8s      *k8s.Client
	kvrocks  *kvrocks.Client
	log      logr.Logger
	password string
	stsNodes []*kvrocks.Node
	requeue  bool
	key      types.NamespacedName
}

func NewKVRocksStandardHandler(
	k8s *k8s.Client,
	kvrocks *kvrocks.Client,
	log logr.Logger,
	key types.NamespacedName,
	instance *kvrocksv1alpha1.KVRocks) *KVRocksStandardHandler {
	return &KVRocksStandardHandler{
		instance: instance,
		k8s:      k8s,
		kvrocks:  kvrocks,
		log:      log,
		requeue:  false,
		key:      key,
	}
}

func (h *KVRocksStandardHandler) Handle() (error, bool) {
	err := h.ensureKubernetes()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.ensureKVRocksStatus()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.resizeStatefulSet()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.cleanPersistentVolumeClaim()
	if err != nil || h.requeue {
		return err, false
	}
	return nil, true
}

func (h *KVRocksStandardHandler) Requeue() bool {
	return h.requeue
}

func (h *KVRocksStandardHandler) Finializer() error {
	if !h.instance.Spec.EnableSentinel {
		return nil
	}
	commHandler := common.NewCommandHandler(h.instance, h.k8s, h.kvrocks, h.password)
	_, masterName := resources.ParseRedisName(h.instance.Name)
	requeue, err := commHandler.RemoveMonitor(masterName)
	h.requeue = requeue
	if err != nil {
		return err
	}
	h.log.Info("sentinel clean up")
	return nil
}
