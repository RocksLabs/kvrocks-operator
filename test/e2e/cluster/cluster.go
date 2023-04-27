package cluster

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
	"github.com/RocksLabs/kvrocks-operator/test/e2e/suite"
)

var _ = Describe("KVRocks cluster controller", func() {
	const (
		timeout  = time.Minute * 10
		interval = time.Second * 10
	)
	var err error
	instance := &kvrocksv1alpha1.KVRocks{}
	dec := k8syaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err = dec.Decode([]byte(suite.KVRocksCluster), nil, instance)
	Expect(err).To(Succeed())
	key := types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName(),
	}
	It("test create kvrocks cluster", func() {
		//	Skip("")
		Expect(suite.K8sClient.Create(suite.CTX, instance)).Should(Succeed())
		Eventually(func() error {
			if err := suite.K8sClient.Get(suite.CTX, key, instance); err != nil {
				return err
			}
			if instance.Status.Status != kvrocksv1alpha1.StatusRunning {
				return errors.New("kvrocks doesn't reach running status")
			}
			if !controllerutil.ContainsFinalizer(instance, kvrocksv1alpha1.KVRocksFinalizer) {
				return errors.New("kvrocks finalizer doesn't exists")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkRedisCluster(key)
		}, timeout, interval).Should(Succeed())
	})

	It("test update kvrocks config", func() {
		// Skip("")
		instance.Spec.KVRocksConfig["slowlog-log-slower-than"] = "250000"
		instance.Spec.KVRocksConfig["profiling-sample-record-threshold-ms"] = "200"
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
		Eventually(func() error {
			return checkRedisCluster(key)
		}, timeout, interval).Should(Succeed())
	})

	It("test change kvrocks password", func() {
		//Skip("")
		Expect(suite.K8sClient.Get(suite.CTX, key, instance)).Should(Succeed())
		instance.Spec.Password = "39c5bb"
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
		Eventually(func() error {
			return checkRedisCluster(key)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when slave down", func() {
		//Skip("")
		for index := 0; index < int(instance.Spec.Master); index++ {
			var pod corev1.Pod
			key := types.NamespacedName{
				Namespace: instance.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d-%d", instance.GetName(), index, 1),
			}
			Expect(suite.K8sClient.Get(suite.CTX, key, &pod)).Should(Succeed())
			Expect(instance.Status.Topo[index].Topology[1].Role).Should(Equal(kvrocks.RoleSlaver))
			Expect(suite.K8sClient.Delete(suite.CTX, &pod)).Should(Succeed())
		}

		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			instance, err = getInstance(key)
			Expect(err).Should(Succeed())
			for index := 0; index < int(instance.Spec.Master); index++ {
				var pod corev1.Pod
				key := types.NamespacedName{
					Namespace: instance.GetNamespace(),
					Name:      fmt.Sprintf("%s-%d-%d", instance.GetName(), index, 1),
				}
				if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
					return err
				}
				if pod.Status.Phase != corev1.PodRunning {
					return errors.New("please wait pod running")
				}
				if instance.Status.Topo[index].Topology[1].Role != kvrocks.RoleSlaver {
					return errors.New("role is incorrect")
				}
				if instance.Status.Topo[index].Topology[1].Failover {
					return errors.New("wait failover over")
				}
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkRedisCluster(key)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when master down", func() {
		for index := 0; index < int(instance.Spec.Master); index++ {
			var pod corev1.Pod
			key := types.NamespacedName{
				Namespace: instance.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d-%d", instance.GetName(), index, 0),
			}
			Expect(suite.K8sClient.Get(suite.CTX, key, &pod)).Should(Succeed())
			Expect(instance.Status.Topo[index].Topology[0].Role).Should(Equal(kvrocks.RoleMaster))
			Expect(suite.K8sClient.Delete(suite.CTX, &pod)).Should(Succeed())
		}

		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			instance, err = getInstance(key)
			Expect(err).Should(Succeed())
			for index := 0; index < int(instance.Spec.Master); index++ {
				var pod corev1.Pod
				key := types.NamespacedName{
					Namespace: instance.GetNamespace(),
					Name:      fmt.Sprintf("%s-%d-%d", instance.GetName(), index, 0),
				}
				if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
					return err
				}
				if pod.Status.Phase != corev1.PodRunning {
					return errors.New("please wait pod running")
				}
				if instance.Status.Topo[index].Topology[0].Role != kvrocks.RoleSlaver {
					return errors.New("role is incorrect")
				}
				if instance.Status.Topo[index].Topology[0].Failover {
					return errors.New("wait failover over")
				}
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkRedisCluster(key)
		}, timeout, interval).Should(Succeed())
	})

	It("test shrink", func() {
		instance, err := getInstance(key)
		Expect(err).Should(Succeed())
		instance.Spec.Replicas = 1
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
		for index := 0; index < int(instance.Spec.Master); index++ {
			key := types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      fmt.Sprintf("%s-%d", instance.GetName(), index),
			}
			Eventually(func() error {
				var sts kruise.StatefulSet
				Expect(suite.K8sClient.Get(suite.CTX, key, &sts)).Should(Succeed())
				if sts.Status.ReadyReplicas != int32(1) {
					return errors.New("ready replicas error")
				}
				if len(sts.Spec.ReserveOrdinals) != 1 || sts.Spec.ReserveOrdinals[0] != 0 {
					return errors.New("ordinals error")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		}
	})

	It("test expansion", func() {
		instance, err := getInstance(key)
		Expect(err).Should(Succeed())
		instance.Spec.Replicas = 2
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
		for index := 0; index < int(instance.Spec.Master); index++ {
			key := types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      fmt.Sprintf("%s-%d", instance.GetName(), index),
			}
			Eventually(func() error {
				var sts kruise.StatefulSet
				Expect(suite.K8sClient.Get(suite.CTX, key, &sts)).Should(Succeed())
				if sts.Status.ReadyReplicas != 2 {
					return errors.New("replication error")
				}
				if len(sts.Spec.ReserveOrdinals) != 0 {
					return errors.New("ordinals error")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		}
		Eventually(func() error {
			return checkRedisCluster(key)
		}, timeout, interval).Should(Succeed())
	})

	It("test clean up", func() {
		Expect(suite.K8sClient.Delete(suite.CTX, instance)).Should(Succeed())
		Eventually(func() bool {
			return k8serr.IsNotFound(suite.K8sClient.Get(suite.CTX, key, instance))
		}, timeout, interval).Should(Equal(true))
	})
})

func getInstance(key types.NamespacedName) (*kvrocksv1alpha1.KVRocks, error) {
	instance := &kvrocksv1alpha1.KVRocks{}
	if err := suite.K8sClient.Get(suite.CTX, key, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

func checkRedisCluster(key types.NamespacedName) error {
	instance, err := getInstance(key)
	if err != nil {
		return err
	}
	slots := []int{}
	masterIP := []string{}
	password := instance.Spec.Password
	version := instance.Status.Version
	for _, partition := range instance.Status.Topo {
		for _, topo := range partition.Topology {
			var pod corev1.Pod
			key := types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      topo.Pod,
			}

			if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
				return err
			}
			node, err := suite.KVRocksClient.ClusterNodeInfo(pod.Status.PodIP, password)
			if err != nil {
				return err
			}
			nodeVersion, err := suite.KVRocksClient.ClusterVersion(topo.Ip, password)
			if err != nil {
				return err
			}
			if version != nodeVersion {
				return fmt.Errorf("version is incorrect, expect: %d, actual: %d", version, nodeVersion)
			}
			if topo.Role != node.Role {
				return fmt.Errorf("role is incorrect, expect: %s, actual: %s", topo.Role, node.Role)
			}
			if topo.Role == kvrocks.RoleMaster {
				masterIP = append(masterIP, topo.Ip)
			}
			if topo.Ip != node.IP {
				return fmt.Errorf("ip is incorrect, expect: %s, actual: %s", topo.Ip, node.IP)
			}
			if !reflect.DeepEqual(topo.Slots, kvrocks.SlotsToString(node.Slots)) {
				return fmt.Errorf("slots is incorrect, expect: %v, actual: %v", topo.Slots, node.Slots)
			} else {
				slots = append(slots, node.Slots...)
			}
			if topo.NodeId != node.NodeId {
				return fmt.Errorf("nodeID is incorrect, expect: %s, actual: %s", topo.NodeId, node.NodeId)
			}
			if topo.MasterId != node.Master {
				return fmt.Errorf("masterID is incorrect, expect: %s, actual: %s", topo.MasterId, node.Master)
			}
			for k, v := range instance.Spec.KVRocksConfig {
				curValue, err := suite.KVRocksClient.GetConfig(node.IP, password, k)
				if err != nil {
					return err
				}
				if *curValue != v {
					return fmt.Errorf("kvrocks config is incorrect, expect: %s, actual: %s", v, *curValue)
				}
			}
		}
	}
	if len(slots) != kvrocks.HashSlotCount {
		return fmt.Errorf("slots total is incorrect, expect: %d, actual: %d", kvrocks.HashSlotCount, len(slots))
	}
	sentinel := resources.GetSentinelInstance(instance)
	for index := 0; index < int(sentinel.Spec.Replicas); index++ {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: sentinel.Namespace,
			Name:      fmt.Sprintf("%s-%d", sentinel.Name, index),
		}
		if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
			return err
		}
		for i := 0; i < int(instance.Spec.Master); i++ {
			_, name := resources.ParseRedisName(instance.Name)
			name = fmt.Sprintf("%s-%d", name, i)
			master, err := suite.KVRocksClient.GetMasterFromSentinel(pod.Status.PodIP, sentinel.Spec.Password, name)
			if err != nil {
				return err
			}
			if master != masterIP[i] {
				return fmt.Errorf("sentinel-%d  monitor master error message,masterIp expect: %s, actual: %s", index, masterIP[i], master)
			}
		}
	}
	pvcSum := int(instance.Spec.Master) * int(instance.Spec.Replicas)
	var pvcList corev1.PersistentVolumeClaimList
	if err := suite.K8sClient.List(suite.CTX, &pvcList, client.InNamespace(instance.Namespace), client.MatchingLabels(instance.Labels)); err != nil {
		return err
	}
	if len(pvcList.Items) != pvcSum {
		return fmt.Errorf("number of pvc is incorrent, expect: %d, actual: %d", pvcSum, len(pvcList.Items))
	}
	return nil
}
