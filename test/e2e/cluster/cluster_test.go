package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
	. "github.com/RocksLabs/kvrocks-operator/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	env           *KubernetesEnv
	ctx           context.Context
	kvrocksClient kvrocks.Client
)

var _ = BeforeSuite(func() {
	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	if configFilePath == "" {
		configFilePath = "../config/config.yaml"
	}
	config, err := NewConfig(configFilePath)
	Expect(err).Should(Succeed())
	env = Start(config)
	ctx = context.Background()
	kvrocksClient = kvrocks.NewKVRocksClient(ctrl.Log)
})

var _ = AfterSuite(func() {
	err := env.Clean()
	Expect(err).Should(Succeed())
})

var _ = Describe("Operator for Cluster Mode", func() {
	const (
		timeout  = time.Minute * 10
		interval = time.Second * 10
	)

	var (
		kvrocksInstance  *kvrocksv1alpha1.KVRocks
		sentinelInstance *kvrocksv1alpha1.KVRocks
		kvrocksKey       types.NamespacedName
		sentinelKey      types.NamespacedName
	)

	BeforeEach(func() {
		var err error
		kvrocksInstance, err = env.ParseManifest(kvrocksv1alpha1.ClusterType)
		Expect(err).Should(Succeed())
		sentinelInstance, err = env.ParseManifest(kvrocksv1alpha1.SentinelType)
		Expect(err).Should(Succeed())

		kvrocksKey = types.NamespacedName{
			Name:      kvrocksInstance.GetName(),
			Namespace: kvrocksInstance.GetNamespace(),
		}

		sentinelKey = types.NamespacedName{
			Name:      sentinelInstance.GetName(),
			Namespace: sentinelInstance.GetNamespace(),
		}

		Expect(env.Client.Create(ctx, kvrocksInstance)).Should(Succeed())
		Expect(env.Client.Create(ctx, sentinelInstance)).Should(Succeed())
		Eventually(func() error {
			if err = env.Client.Get(ctx, kvrocksKey, kvrocksInstance); err != nil {
				return err
			}
			if kvrocksInstance.Status.Status != kvrocksv1alpha1.StatusRunning {
				return errors.New("kvrocks doesn't reach running status")
			}
			if !controllerutil.ContainsFinalizer(kvrocksInstance, kvrocksv1alpha1.KVRocksFinalizer) {
				return errors.New("kvrocks doesn't contain finalizer")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		err := env.Client.Get(ctx, kvrocksKey, kvrocksInstance)
		Expect(err).Should(Succeed())
		Expect(env.Client.Delete(ctx, kvrocksInstance)).Should(Succeed())
		Eventually(func() bool {
			return k8serr.IsNotFound(env.Client.Get(ctx, kvrocksKey, kvrocksInstance))
		}, timeout, interval).Should(Equal(true))

		err = env.Client.Get(ctx, sentinelKey, sentinelInstance)
		Expect(err).Should(Succeed())
		Expect(env.Client.Delete(ctx, sentinelInstance)).Should(Succeed())
		Eventually(func() bool {
			return k8serr.IsNotFound(env.Client.Get(ctx, sentinelKey, sentinelInstance))
		}, timeout, interval).Should(Equal(true))
	})

	It("test update kvrocks config", func() {
		kvrocksInstance.Spec.KVRocksConfig["slowlog-log-slower-than"] = "250000"
		kvrocksInstance.Spec.KVRocksConfig["profiling-sample-record-threshold-ms"] = "200"
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})

	It("test change kvrocks password", func() {
		Expect(env.Client.Get(ctx, kvrocksKey, kvrocksInstance)).Should(Succeed())
		kvrocksInstance.Spec.Password = "39c5bb"
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when slave down", func() {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: kvrocksInstance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d-%d", kvrocksInstance.GetName(), 0, 1),
		}
		Expect(env.Client.Get(ctx, key, &pod)).Should(Succeed())
		Expect(kvrocksInstance.Status.Topo[0].Topology[1].Role).Should(Equal(kvrocks.RoleSlaver))
		Expect(env.Client.Delete(ctx, &pod)).Should(Succeed())

		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			err := env.Client.Get(ctx, kvrocksKey, kvrocksInstance)
			if err != nil {
				return err
			}

			var pod corev1.Pod
			key := types.NamespacedName{
				Namespace: kvrocksInstance.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d-%d", kvrocksInstance.GetName(), 0, 1),
			}
			if err := env.Client.Get(ctx, key, &pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return errors.New("please wait pod running")
			}
			if kvrocksInstance.Status.Topo[0].Topology[1].Failover {
				return errors.New("wait failover over")
			}

			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when master down", func() {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: kvrocksInstance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d-%d", kvrocksInstance.GetName(), 0, 0),
		}
		Expect(env.Client.Get(ctx, key, &pod)).Should(Succeed())
		Expect(kvrocksInstance.Status.Topo[0].Topology[0].Role).Should(Equal(kvrocks.RoleMaster))
		Expect(env.Client.Delete(ctx, &pod)).Should(Succeed())

		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			err := env.Client.Get(ctx, kvrocksKey, kvrocksInstance)
			if err != nil {
				return err
			}
			Expect(err).Should(Succeed())

			var pod corev1.Pod
			key := types.NamespacedName{
				Namespace: kvrocksInstance.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d-%d", kvrocksInstance.GetName(), 0, 0),
			}
			if err := env.Client.Get(ctx, key, &pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return errors.New("please wait pod running")
			}
			if kvrocksInstance.Status.Topo[0].Topology[0].Failover {
				return errors.New("wait failover over")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})

	It("test shrink", func() {
		kvrocksInstance.Spec.Replicas = 1
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		for index := 0; index < int(kvrocksInstance.Spec.Master); index++ {
			key := types.NamespacedName{
				Namespace: kvrocksInstance.Namespace,
				Name:      fmt.Sprintf("%s-%d", kvrocksInstance.GetName(), index),
			}
			Eventually(func() error {
				var sts kruise.StatefulSet
				Expect(env.Client.Get(ctx, key, &sts)).Should(Succeed())
				if sts.Status.ReadyReplicas != int32(1) {
					return errors.New("ready replicas error")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		}
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})

	It("test expansion", func() {
		kvrocksInstance.Spec.Replicas = 3
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		for index := 0; index < int(kvrocksInstance.Spec.Master); index++ {
			key := types.NamespacedName{
				Namespace: kvrocksInstance.Namespace,
				Name:      fmt.Sprintf("%s-%d", kvrocksInstance.GetName(), index),
			}
			Eventually(func() error {
				var sts kruise.StatefulSet
				Expect(env.Client.Get(ctx, key, &sts)).Should(Succeed())
				if sts.Status.ReadyReplicas != 3 {
					return errors.New("replication error")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		}
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})

	It("test expansion and shrink master", func() {
		// expansion
		kvrocksInstance.Spec.Master = 5
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		time.Sleep(time.Second * 30)
		for index := 0; index < int(kvrocksInstance.Spec.Master); index++ {
			key := types.NamespacedName{
				Namespace: kvrocksInstance.Namespace,
				Name:      fmt.Sprintf("%s-%d", kvrocksInstance.GetName(), index),
			}
			Eventually(func() error {
				var sts kruise.StatefulSet
				Expect(env.Client.Get(ctx, key, &sts)).Should(Succeed())
				if sts.Status.ReadyReplicas != kvrocksInstance.Spec.Replicas {
					return errors.New("waitting ready")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		}
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())

		// shrink
		err := env.Client.Get(ctx, kvrocksKey, kvrocksInstance)
		Expect(err).Should(Succeed())
		kvrocksInstance.Spec.Master = 3
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		time.Sleep(time.Second * 30)
		for index := 0; index < int(kvrocksInstance.Spec.Master); index++ {
			key := types.NamespacedName{
				Namespace: kvrocksInstance.Namespace,
				Name:      fmt.Sprintf("%s-%d", kvrocksInstance.GetName(), index),
			}
			Eventually(func() error {
				var sts kruise.StatefulSet
				Expect(env.Client.Get(ctx, key, &sts)).Should(Succeed())
				if sts.Status.ReadyReplicas != kvrocksInstance.Spec.Replicas {
					return errors.New("waitting ready")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		}
		Eventually(func() error {
			return checkKvrocksCluster(kvrocksKey, sentinelKey)
		}, timeout, interval).Should(Succeed())
	})
})

func checkKvrocksCluster(kvrocksKey, sentinelKey types.NamespacedName) error {
	kvrocksInstance := &kvrocksv1alpha1.KVRocks{}
	err := env.Client.Get(ctx, kvrocksKey, kvrocksInstance)
	if err != nil {
		return err
	}
	sentinelInstance := &kvrocksv1alpha1.KVRocks{}
	err = env.Client.Get(ctx, sentinelKey, sentinelInstance)
	if err != nil {
		return err
	}
	slots := []int{}
	masterIP := []string{}
	password := kvrocksInstance.Spec.Password
	for _, partition := range kvrocksInstance.Status.Topo {
		for _, topo := range partition.Topology {
			var pod corev1.Pod
			key := types.NamespacedName{
				Namespace: kvrocksInstance.GetNamespace(),
				Name:      topo.Pod,
			}
			if err := env.Client.Get(ctx, key, &pod); err != nil {
				return err
			}
			node, err := kvrocksClient.ClusterNodeInfo(pod.Status.PodIP, password)
			if err != nil {
				return err
			}
			if topo.Role != node.Role {
				return fmt.Errorf("role is incorrect, expect: %s, actual: %s", topo.Role, node.Role)
			}
			if topo.Role == kvrocks.RoleMaster {
				masterIP = append(masterIP, topo.Ip)
				if !reflect.DeepEqual(topo.Slots, kvrocks.SlotsToString(node.Slots)) {
					return fmt.Errorf("slots is incorrect, expect: %v, actual: %v", topo.Slots, node.Slots)
				} else {
					slots = append(slots, node.Slots...)
				}
			}
			if topo.Ip != node.IP {
				return fmt.Errorf("ip is incorrect, expect: %s, actual: %s", topo.Ip, node.IP)
			}
			if topo.NodeId != node.NodeId {
				return fmt.Errorf("nodeID is incorrect, expect: %s, actual: %s", topo.NodeId, node.NodeId)
			}
			if topo.MasterId != node.Master {
				return fmt.Errorf("masterID is incorrect, expect: %s, actual: %s", topo.MasterId, node.Master)
			}
			for k, v := range kvrocksInstance.Spec.KVRocksConfig {
				curValue, err := kvrocksClient.GetConfig(node.IP, password, k)
				if err != nil {
					return err
				}
				if *curValue != v {
					return fmt.Errorf("kvrocks config is incorrect, expect: %s, actual: %s", v, *curValue)
				}
			}
		}
	}
	if len(slots) != kvrocks.MaxSlotID+1 {
		return fmt.Errorf("slots total is incorrect, expect: %d, actual: %d", (kvrocks.MaxSlotID + 1), len(slots))
	}

	podList, err := getSentinelPodList(sentinelInstance)
	if err != nil {
		return fmt.Errorf("get sentinel pod list error: %v", err)
	}
	for _, pod := range podList.Items {
		for i := 0; i < int(kvrocksInstance.Spec.Master); i++ {
			_, name := resources.ParseRedisName(kvrocksInstance.Name)
			name = fmt.Sprintf("%s-%d", name, i)
			master, err := kvrocksClient.GetMasterFromSentinel(pod.Status.PodIP, sentinelInstance.Spec.Password, name)
			if err != nil {
				return err
			}
			if master != masterIP[i] {
				return fmt.Errorf("sentinel-1  monitor master error message,masterIp expect: %s, actual: %s", masterIP[i], master)
			}
		}
	}

	pvcSum := int(kvrocksInstance.Spec.Master) * int(kvrocksInstance.Spec.Replicas)
	var pvcList corev1.PersistentVolumeClaimList
	if err := env.Client.List(ctx, &pvcList, client.InNamespace(kvrocksInstance.Namespace), client.MatchingLabels(kvrocksInstance.Labels)); err != nil {
		return err
	}
	if len(pvcList.Items) != pvcSum {
		return fmt.Errorf("number of pvc is incorrent, expect: %d, actual: %d", pvcSum, len(pvcList.Items))
	}
	return nil
}

func getSentinelPodList(sentinel *kvrocksv1alpha1.KVRocks) (*corev1.PodList, error) {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{
		Namespace: sentinel.Namespace,
		Name:      sentinel.Name,
	}

	if err := env.Client.Get(ctx, key, deployment); err != nil {
		return nil, err
	}

	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector()
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(sentinel.Namespace),
		client.MatchingLabelsSelector{Selector: labelSelector},
	}
	if err := env.Client.List(ctx, podList, listOpts...); err != nil {
		return nil, err
	}
	return podList, nil
}
