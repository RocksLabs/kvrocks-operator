package standard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
	. "github.com/RocksLabs/kvrocks-operator/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	env           *KubernetesEnv
	ctx           context.Context
	kvrocksClient *kvrocks.Client
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

var _ = Describe("Operator for Standard Mode", func() {
	const (
		timeout  = time.Minute * 10
		interval = time.Second * 10
	)

	var (
		instance *kvrocksv1alpha1.KVRocks
		key      types.NamespacedName
	)

	BeforeEach(func() {
		var err error
		instance, err = env.ParseManifest(kvrocksv1alpha1.StandardType)
		Expect(err).Should(Succeed())

		key = types.NamespacedName{
			Namespace: instance.GetNamespace(),
			Name:      instance.GetName(),
		}

		Expect(env.Client.Create(ctx, instance)).Should(Succeed())
		Eventually(func() error {
			if err = env.Client.Get(ctx, key, instance); err != nil {
				return err
			}
			if instance.Status.Status != kvrocksv1alpha1.StatusRunning {
				return errors.New("kvrocks doesn't reach running status")
			}
			if !controllerutil.ContainsFinalizer(instance, kvrocksv1alpha1.KVRocksFinalizer) {
				return errors.New("kvrocks doesn't contain finalizer")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		Expect(env.Client.Delete(ctx, instance)).Should(Succeed())
		Eventually(func() bool {
			return k8serr.IsNotFound(env.Client.Get(ctx, key, instance))
		}, timeout, interval).Should(Equal(true))
	})

	It("test update kvrocks config", func() {
		instance.Spec.KVRocksConfig["slowlog-log-slower-than"] = "250000"
		instance.Spec.KVRocksConfig["profiling-sample-record-threshold-ms"] = "200"
		Expect(env.Client.Update(ctx, instance)).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test change password", func() {
		instance.Spec.Password = "39c5bb"
		Expect(env.Client.Update(ctx, instance)).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when slave down", func() {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: instance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d", instance.GetName(), 1), // TODO remove 1
		}
		Expect(env.Client.Get(ctx, key, &pod)).Should(Succeed())
		Expect(pod.Labels[resources.RedisRole]).Should(Equal(kvrocks.RoleSlaver))
		Expect(env.Client.Delete(ctx, &pod)).Should(Succeed())

		time.Sleep(time.Second * 30)
		Eventually(func() error {
			if err := env.Client.Get(ctx, key, &pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return errors.New("please wait pod running")
			}
			if pod.Labels[resources.RedisRole] != kvrocks.RoleSlaver {
				return fmt.Errorf("role is incorrect, expect: %s, actual: %s", kvrocks.RoleSlaver, pod.Labels[resources.RedisRole])
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when master down", func() {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: instance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d", instance.GetName(), 0), // TODO remove 0
		}
		Expect(env.Client.Get(ctx, key, &pod)).Should(Succeed())
		Expect(pod.Labels[resources.RedisRole]).Should(Equal(kvrocks.RoleMaster))
		Expect(env.Client.Delete(ctx, &pod)).Should(Succeed())
		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			if err := env.Client.Get(ctx, key, &pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return errors.New("please wait pod running")
			}
			if pod.Labels[resources.RedisRole] != kvrocks.RoleSlaver {
				return fmt.Errorf("role is incorrect, expect: %s, actual: %s", kvrocks.RoleSlaver, pod.Labels[resources.RedisRole])
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when sentinel down", func() {
		sentinel := resources.GetSentinelInstance(instance)
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: instance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d", sentinel.GetName(), 0),
		}
		Expect(env.Client.Get(ctx, key, &pod)).Should(Succeed())
		Expect(env.Client.Delete(ctx, &pod)).Should(Succeed())
		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			if err := env.Client.Get(ctx, key, &pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return errors.New("please wait pod running")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test shrink", func() {
		Expect(env.Client.Get(ctx, key, instance)).Should(Succeed())
		// pod xx-1 should be reserved
		instance.Spec.Replicas = 1
		Expect(env.Client.Update(ctx, instance)).Should(Succeed())
		var sts kruise.StatefulSet
		Eventually(func() error {
			Expect(env.Client.Get(ctx, key, &sts)).Should(Succeed())
			if sts.Status.ReadyReplicas != int32(1) {
				return errors.New("ready replicas error")
			}
			if len(sts.Spec.ReserveOrdinals) != 0 {
				return errors.New("ordinals error")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test expansion", func() {
		Expect(env.Client.Get(ctx, key, instance)).Should(Succeed())
		instance.Spec.Replicas = 5
		Expect(env.Client.Update(ctx, instance)).Should(Succeed())
		Eventually(func() error {
			var sts kruise.StatefulSet
			Expect(env.Client.Get(ctx, key, &sts)).Should(Succeed())
			if sts.Status.ReadyReplicas != 5 {
				return errors.New("replication error")
			}
			if len(sts.Spec.ReserveOrdinals) != 0 {
				return errors.New("ordinals error")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

})

func checkKVRocks(instance *kvrocksv1alpha1.KVRocks) error {
	password := instance.Spec.Password
	replicas := int(instance.Spec.Replicas)
	masterIP := []string{}
	masterOfSlave := map[int]string{}

	for index := 0; index < replicas; index++ {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: instance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d", instance.GetName(), index),
		}
		if err := env.Client.Get(ctx, key, &pod); err != nil {
			return err
		}

		node, err := kvrocksClient.NodeInfo(pod.Status.PodIP, password)
		if err != nil {
			return err
		}
		if node.Role != pod.Labels[resources.RedisRole] {
			return fmt.Errorf("reole label is incorrect,expect: %s, actual: %s", node.Role, pod.Labels[resources.RedisRole])
		}
		if node.Role == kvrocks.RoleMaster {
			masterIP = append(masterIP, node.IP)
		}
		if node.Role == kvrocks.RoleSlaver {
			curMaster, err := kvrocksClient.GetMaster(node.IP, password)
			if err != nil {
				return err
			}
			masterOfSlave[index] = curMaster
		}
		for k, v := range instance.Spec.KVRocksConfig {
			curValue, err := kvrocksClient.GetConfig(node.IP, password, k)
			if err != nil {
				return err
			}
			if *curValue != v {
				return fmt.Errorf("kvrocks config is incorrect, expect: %s, actual: %s", v, *curValue)
			}
		}
	}
	if len(masterIP) != 1 {
		return fmt.Errorf("wrong number of master,masters: %v", masterIP)
	}
	for index, curMasterIP := range masterOfSlave {
		if curMasterIP != masterIP[0] {
			return fmt.Errorf("slave %d has wrong master, expect: %s, actual: %s", index, masterIP[0], curMasterIP)
		}
	}

	sentinel := resources.GetSentinelInstance(instance)
	for index := 0; index < int(sentinel.Spec.Replicas); index++ {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: sentinel.Namespace,
			Name:      fmt.Sprintf("%s-%d", sentinel.Name, index),
		}
		if err := env.Client.Get(ctx, key, &pod); err != nil {
			return err
		}

		_, name := resources.ParseRedisName(instance.Name)
		master, err := kvrocksClient.GetMasterFromSentinel(pod.Status.PodIP, sentinel.Spec.Password, name)
		if err != nil {
			return err
		}
		if master != masterIP[0] {
			return fmt.Errorf("sentinel-%d  monitor master error message,masterIp expect: %s, actual: %s", index, masterIP[0], master)
		}
	}

	var pvcList corev1.PersistentVolumeClaimList
	if err := env.Client.List(ctx, &pvcList, client.InNamespace(instance.Namespace), client.MatchingLabels(instance.Labels)); err != nil {
		return err
	}
	if len(pvcList.Items) != replicas {
		return fmt.Errorf("number of pvc is incorrent, expect: %d, actual: %d", replicas, len(pvcList.Items))
	}
	return nil
}
