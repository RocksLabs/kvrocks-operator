package cluster

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/k8s"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/controllers/common"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/resources"
)

type KVRocksClusterHandler struct {
	instance *kvrocksv1alpha1.KVRocks
	k8s      *k8s.Client
	kvrocks  *kvrocks.Client
	log      logr.Logger
	password string
	requeue  bool
	stsNodes [][]*kvrocks.Node
	key      types.NamespacedName
	version  int
	masters  map[string]*kvrocks.Node
}

func NewKVRocksClusterHandler(
	k8s *k8s.Client,
	kvrocks *kvrocks.Client,
	log logr.Logger,
	key types.NamespacedName,
	instance *kvrocksv1alpha1.KVRocks) *KVRocksClusterHandler {
	return &KVRocksClusterHandler{
		instance: instance,
		k8s:      k8s,
		kvrocks:  kvrocks,
		log:      log,
		requeue:  false,
		key:      key,
	}
}

func (h *KVRocksClusterHandler) Handle() (error, bool) {
	if h.instance.Status.Shrink != nil {
		err := h.cleanStatefulSet()
		if err != nil || h.requeue {
			return err, false
		}
	}
	err := h.ensureKubernetes()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.ensureFailover()
	if err != nil || h.requeue {
		return err, false
	}

	err = h.ensureKVRocksStatus()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.reBalance()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.ensureShrink()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.ensureSentinel()
	if err != nil || h.requeue {
		return err, false
	}
	err = h.cleanPersistentVolumeClaim()
	if err != nil || h.requeue {
		return err, false
	}
	return nil, true
}

func (h *KVRocksClusterHandler) Requeue() bool {
	return h.requeue
}

func (h *KVRocksClusterHandler) Finializer() error {
	commHandler := common.NewCommandHandler(h.instance, h.k8s, h.kvrocks, h.password)
	_, masterName := resources.ParseRedisName(h.instance.Name)
	for index := 0; index < int(h.instance.Spec.Master); index++ {
		requeue, err := commHandler.RemoveMonitor(masterName, index)
		h.requeue = requeue
		if err != nil {
			return err
		}
	}
	h.log.Info("sentinel clean up")
	return nil
}
