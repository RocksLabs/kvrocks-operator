package standard

import (
	"errors"
	"fmt"
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

	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/resources"
	"github.com/KvrocksLabs/kvrocks-operator/test/e2e/suite"
)

var _ = Describe("KVRocks standard controller", func() {
	const (
		timeout  = time.Minute * 10
		interval = time.Second * 10
	)
	var err error
	instance := &kvrocksv1alpha1.KVRocks{}
	dec := k8syaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err = dec.Decode([]byte(suite.KVRocksStandard), nil, instance)
	Expect(err).To(Succeed())
	key := types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName(),
	}

	It("test create kvrocks standard", func() {
		Expect(suite.K8sClient.Create(suite.CTX, instance)).Should(Succeed())
		Eventually(func() error {
			if err = suite.K8sClient.Get(suite.CTX, key, instance); err != nil {
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
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test update kvrocks config", func() {
		Expect(suite.K8sClient.Get(suite.CTX, key, instance)).Should(Succeed())
		instance.Spec.KVRocksConfig["slowlog-log-slower-than"] = "250000"
		instance.Spec.KVRocksConfig["profiling-sample-record-threshold-ms"] = "200"
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test change password", func() {
		Expect(suite.K8sClient.Get(suite.CTX, key, instance)).Should(Succeed())
		instance.Spec.Password = "39c5bb"
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test recover when slave down", func() {
		var pod corev1.Pod
		key := types.NamespacedName{
			Namespace: instance.GetNamespace(),
			Name:      fmt.Sprintf("%s-%d", instance.GetName(), 1),
		}
		Expect(suite.K8sClient.Get(suite.CTX, key, &pod)).Should(Succeed())
		Expect(pod.Labels[resources.RedisRole]).Should(Equal(kvrocks.RoleSlaver))
		Expect(suite.K8sClient.Delete(suite.CTX, &pod)).Should(Succeed())
		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
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
			Name:      fmt.Sprintf("%s-%d", instance.GetName(), 0),
		}
		Expect(suite.K8sClient.Get(suite.CTX, key, &pod)).Should(Succeed())
		Expect(pod.Labels[resources.RedisRole]).Should(Equal(kvrocks.RoleMaster))
		Expect(suite.K8sClient.Delete(suite.CTX, &pod)).Should(Succeed())
		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
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
		Expect(suite.K8sClient.Get(suite.CTX, key, &pod)).Should(Succeed())
		Expect(suite.K8sClient.Delete(suite.CTX, &pod)).Should(Succeed())
		// wait pod reconstruction
		time.Sleep(time.Second * 30)
		Eventually(func() error {
			if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
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
		Expect(suite.K8sClient.Get(suite.CTX, key, instance)).Should(Succeed())
		// pod xx-1 should be reserved
		instance.Spec.Replicas = 1
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
		var sts kruise.StatefulSet
		Eventually(func() error {
			Expect(suite.K8sClient.Get(suite.CTX, key, &sts)).Should(Succeed())
			if sts.Status.ReadyReplicas != int32(1) {
				return errors.New("ready replicas error")
			}
			if len(sts.Spec.ReserveOrdinals) != 1 || sts.Spec.ReserveOrdinals[0] != 0 {
				return errors.New("ordinals error")
			}
			return nil
		}, timeout, interval).Should(Succeed())

	})

	It("test expansion", func() {
		Expect(suite.K8sClient.Get(suite.CTX, key, instance)).Should(Succeed())
		instance.Spec.Replicas = 2
		Expect(suite.K8sClient.Update(suite.CTX, instance)).Should(Succeed())
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
		Eventually(func() error {
			return checkKVRocks(instance)
		}, timeout, interval).Should(Succeed())
	})

	It("test clean up", func() {
		Expect(suite.K8sClient.Delete(suite.CTX, instance)).Should(Succeed())
		Eventually(func() bool {
			return k8serr.IsNotFound(suite.K8sClient.Get(suite.CTX, key, instance))
		}, timeout, interval).Should(Equal(true))
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
			Namespace: instance.Namespace,
			Name:      fmt.Sprintf("%s-%d", instance.Name, index),
		}
		if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
			return err
		}
		node, err := suite.KVRocksClient.NodeInfo(pod.Status.PodIP, password)
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
			curMaster, err := suite.KVRocksClient.GetMaster(node.IP, password)
			if err != nil {
				return err
			}
			masterOfSlave[index] = curMaster
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
		if err := suite.K8sClient.Get(suite.CTX, key, &pod); err != nil {
			return err
		}
		_, name := resources.ParseRedisName(instance.Name)
		master, err := suite.KVRocksClient.GetMasterFromSentinel(pod.Status.PodIP, sentinel.Spec.Password, name)
		if err != nil {
			return err
		}
		if master != masterIP[0] {
			return fmt.Errorf("sentinel-%d  monitor master error message,masterIp expect: %s, actual: %s", index, masterIP[0], master)
		}
	}
	var pvcList corev1.PersistentVolumeClaimList
	if err := suite.K8sClient.List(suite.CTX, &pvcList, client.InNamespace(instance.Namespace), client.MatchingLabels(instance.Labels)); err != nil {
		return err
	}
	if len(pvcList.Items) != replicas {
		return fmt.Errorf("number of pvc is incorrent, expect: %d, actual: %d", replicas, len(pvcList.Items))
	}
	return nil
}
