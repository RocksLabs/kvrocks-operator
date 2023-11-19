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
	chaosmeshv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sApiClient "sigs.k8s.io/controller-runtime/pkg/client"
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

	// Chaosmesh: try to kill kvrocks operator every two minutes

	if enabled, _ := env.GetConfig("ChaosMeshEnabled"); enabled.(bool) == true {
		env.ScheduleInjectPodKill(
			chaosmeshv1alpha1.PodSelectorSpec{
				GenericSelectorSpec: chaosmeshv1alpha1.GenericSelectorSpec{
					Namespaces:     []string{config.Namespace},
					LabelSelectors: map[string]string{"app": "kvrocks-operator-controller-manager"},
				},
			},
			"*/2 * * * *",
			chaosmeshv1alpha1.OneMode,
		)
	}
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
		kvrocksInstance  *kvrocksv1alpha1.KVRocks
		sentinelInstance *kvrocksv1alpha1.KVRocks
		kvrocksKey       types.NamespacedName
		sentinelKey      types.NamespacedName
	)

	BeforeEach(func() {
		var err error
		kvrocksInstance, err = env.ParseManifest(kvrocksv1alpha1.StandardType)
		Expect(err).Should(Succeed())
		sentinelInstance, err = env.ParseManifest(kvrocksv1alpha1.SentinelType)
		Expect(err).Should(Succeed())

		kvrocksKey = types.NamespacedName{
			Namespace: kvrocksInstance.GetNamespace(),
			Name:      kvrocksInstance.GetName(),
		}

		sentinelKey = types.NamespacedName{
			Namespace: sentinelInstance.GetNamespace(),
			Name:      sentinelInstance.GetName(),
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
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		Expect(env.Client.Delete(ctx, kvrocksInstance)).Should(Succeed())
		Expect(env.Client.Delete(ctx, sentinelInstance)).Should(Succeed())
		Eventually(func() bool {
			b1 := k8serr.IsNotFound(env.Client.Get(ctx, kvrocksKey, kvrocksInstance))
			b2 := k8serr.IsNotFound(env.Client.Get(ctx, sentinelKey, sentinelInstance))
			return b1 && b2
		}, timeout, interval).Should(Equal(true))
	})

	It("test update kvrocks config", func() {
		kvrocksInstance.Spec.KVRocksConfig["slowlog-log-slower-than"] = "250000"
		kvrocksInstance.Spec.KVRocksConfig["profiling-sample-record-threshold-ms"] = "200"
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})

	It("test change password", func() {
		kvrocksInstance.Spec.Password = "39c5bb"
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when slave down", func() {
		key, pod := getKvrocksByRole(kvrocksInstance, kvrocks.RoleSlaver)
		Expect(env.Client.Delete(ctx, &pod)).Should(Succeed())

		time.Sleep(time.Second * 30)
		Eventually(func() error {
			if err := env.Client.Get(ctx, key, &pod); err != nil {
				return err
			}
			if pod.Status.Phase != corev1.PodRunning {
				return errors.New("please wait pod running")
			}
			if pod.Labels[resources.KvrocksRole] != kvrocks.RoleSlaver {
				return fmt.Errorf("role is incorrect, expect: %s, actual: %s", kvrocks.RoleSlaver, pod.Labels[resources.KvrocksRole])
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when master down", func() {
		key, pod := getKvrocksByRole(kvrocksInstance, kvrocks.RoleMaster)
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
			if pod.Labels[resources.KvrocksRole] != kvrocks.RoleSlaver {
				return fmt.Errorf("role is incorrect, expect: %s, actual: %s", kvrocks.RoleSlaver, pod.Labels[resources.KvrocksRole])
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when sentinel down", func() {
		podList, err := getSentinelPodList(sentinelInstance)
		Expect(err).Should(Succeed())
		Expect(len(podList.Items)).Should(Equal(int(sentinelInstance.Spec.Replicas)))
		Expect(env.Client.Delete(ctx, &podList.Items[0])).Should(Succeed())

		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			podList, err := getSentinelPodList(sentinelInstance)
			if err != nil {
				return err
			}
			if len(podList.Items) != int(sentinelInstance.Spec.Replicas) {
				return fmt.Errorf("please wait pod running")
			}
			for _, pod := range podList.Items {
				if pod.Status.Phase != corev1.PodRunning {
					return errors.New("please wait pod running")
				}
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})

	It("test shrink", func() {
		Expect(env.Client.Get(ctx, kvrocksKey, kvrocksInstance)).Should(Succeed())
		// pod xx-1 should be reserved
		kvrocksInstance.Spec.Replicas = 1
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		var sts kruise.StatefulSet
		Eventually(func() error {
			Expect(env.Client.Get(ctx, kvrocksKey, &sts)).Should(Succeed())
			if sts.Status.ReadyReplicas != int32(1) {
				return errors.New("ready replicas error")
			}
			if len(sts.Spec.ReserveOrdinals) != 0 {
				return errors.New("ordinals error")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})

	It("test expansion", func() {
		Expect(env.Client.Get(ctx, kvrocksKey, kvrocksInstance)).Should(Succeed())
		kvrocksInstance.Spec.Replicas = 5
		Expect(env.Client.Update(ctx, kvrocksInstance)).Should(Succeed())
		Eventually(func() error {
			var sts kruise.StatefulSet
			Expect(env.Client.Get(ctx, kvrocksKey, &sts)).Should(Succeed())
			if sts.Status.ReadyReplicas != 5 {
				return errors.New("replication error")
			}
			if len(sts.Spec.ReserveOrdinals) != 0 {
				return errors.New("ordinals error")
			}
			return nil
		}, timeout, interval).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(kvrocksInstance, sentinelInstance)
		}, timeout, interval).Should(Succeed())
	})
})

func checkKVRocks(kvrocksInstance, sentinelInstance *kvrocksv1alpha1.KVRocks) error {
	password := kvrocksInstance.Spec.Password
	replicas := int(kvrocksInstance.Spec.Replicas)
	masterIP := []string{}
	masterOfSlave := map[int]string{}

	for index := 0; index < replicas; index++ {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: kvrocksInstance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d", kvrocksInstance.GetName(), index),
		}
		if err := env.Client.Get(ctx, key, &pod); err != nil {
			return err
		}

		node, err := kvrocksClient.NodeInfo(pod.Status.PodIP, password)
		if err != nil {
			return err
		}
		if node.Role != pod.Labels[resources.KvrocksRole] {
			return fmt.Errorf("reole label is incorrect,expect: %s, actual: %s", node.Role, pod.Labels[resources.KvrocksRole])
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
	if len(masterIP) != 1 {
		return fmt.Errorf("wrong number of master,masters: %v", masterIP)
	}
	for index, curMasterIP := range masterOfSlave {
		if curMasterIP != masterIP[0] {
			return fmt.Errorf("slave %d has wrong master, expect: %s, actual: %s", index, masterIP[0], curMasterIP)
		}
	}

	podList, err := getSentinelPodList(sentinelInstance)
	if err != nil {
		return fmt.Errorf("get sentinel pod list error: %v", err)
	}
	for _, pod := range podList.Items {
		_, name := resources.ParseRedisName(kvrocksInstance.Name)
		master, err := kvrocksClient.GetMasterFromSentinel(pod.Status.PodIP, sentinelInstance.Spec.Password, name)
		if err != nil {
			return err
		}
		if master != masterIP[0] {
			return fmt.Errorf("sentinel %s  monitor master error message,masterIp expect: %s, actual: %s", pod.Name, masterIP[0], master)
		}
	}

	var pvcList corev1.PersistentVolumeClaimList
	if err := env.Client.List(ctx, &pvcList, k8sApiClient.InNamespace(kvrocksInstance.Namespace), k8sApiClient.MatchingLabels(kvrocksInstance.Labels)); err != nil {
		return err
	}
	if len(pvcList.Items) != replicas {
		return fmt.Errorf("number of pvc is incorrent, expect: %d, actual: %d", replicas, len(pvcList.Items))
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
	listOpts := []k8sApiClient.ListOption{
		k8sApiClient.InNamespace(sentinel.Namespace),
		k8sApiClient.MatchingLabelsSelector{Selector: labelSelector},
	}
	if err := env.Client.List(ctx, podList, listOpts...); err != nil {
		return nil, err
	}
	return podList, nil
}

func getKvrocksByRole(instance *kvrocksv1alpha1.KVRocks, role string) (key types.NamespacedName, pod corev1.Pod) {
	replicas := int(instance.Spec.Replicas)
	for i := 0; i < replicas; i++ {
		key = types.NamespacedName{
			Namespace: instance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d", instance.GetName(), i),
		}
		Expect(env.Client.Get(ctx, key, &pod)).Should(Succeed())
		if pod.Labels[resources.KvrocksRole] == role {
			break
		}
	}
	return
}
