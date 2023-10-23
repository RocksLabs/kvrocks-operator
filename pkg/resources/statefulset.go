package resources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openkruise/kruise-api/apps/pub"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

var TerminationGracePeriodSeconds int64 = 20

const DefaultStorageSize = "10Gi"

func NewStatefulSet(instance *kvrocksv1alpha1.KVRocks, name string) *kruise.StatefulSet {
	labels := MergeLabels(instance.Labels, StatefulSetLabels(name))
	sts := &kruise.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: kruise.StatefulSetSpec{
			Replicas:            &instance.Spec.Replicas,
			PodManagementPolicy: v1.ParallelPodManagement, // 并行启动终止 pod
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			UpdateStrategy: kruise.StatefulSetUpdateStrategy{
				RollingUpdate: &kruise.RollingUpdateStatefulSetStrategy{
					PodUpdatePolicy: kruise.InPlaceIfPossiblePodUpdateStrategyType,
					Paused:          false,
					InPlaceUpdateStrategy: &pub.InPlaceUpdateStrategy{
						GracePeriodSeconds: 10,
					},
				},
				Type: v1.RollingUpdateStatefulSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Affinity: getAffinity(instance, labels),
					// ImagePullSecrets:              instance.Spec.ImagePullSecrets,
					// SecurityContext:               instance.Spec.SecurityContext,
					NodeSelector:                  instance.Spec.NodeSelector,
					Tolerations:                   instance.Spec.Toleration,
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					// SchedulerName:                 instance.Spec.SchedulerName,
					ReadinessGates: []corev1.PodReadinessGate{{
						ConditionType: pub.InPlaceUpdateReady,
					}},
					Volumes: []corev1.Volume{
						{
							Name: "conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, getPersistentClaim(instance, labels))
	return sts
}

func getAffinity(instance *kvrocksv1alpha1.KVRocks, labels map[string]string) *corev1.Affinity {
	affinity := &corev1.Affinity{}
	if instance.Spec.Affinity != nil {
		affinity = instance.Spec.Affinity.DeepCopy()
	}
	if affinity.PodAntiAffinity == nil {
		affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	hostnameTopo := corev1.PodAffinityTerm{
		TopologyKey: "kubernetes.io/hostname",
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
	}
	hostNameTopoWeak := corev1.WeightedPodAffinityTerm{
		Weight:          100,
		PodAffinityTerm: hostnameTopo,
	}
	affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, hostNameTopoWeak)
	return affinity
}

func getPersistentClaim(instance *kvrocksv1alpha1.KVRocks, labels map[string]string) corev1.PersistentVolumeClaim {
	mode := corev1.PersistentVolumeFilesystem
	var class *string = nil
	size := resource.MustParse(DefaultStorageSize)
	if instance.Spec.Storage != nil {
		if instance.Spec.Storage.Class != "" {
			class = &instance.Spec.Storage.Class
		}

		if !instance.Spec.Storage.Size.IsZero() {
			size = instance.Spec.Storage.Size
		}
	}
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "data",
			Labels: labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
			StorageClassName: class,
			VolumeMode:       &mode,
		},
	}
}

func NewSentinelStatefulSet(instance *kvrocksv1alpha1.KVRocks) *kruise.StatefulSet {
	sts := NewStatefulSet(instance, GetStatefulSetName(instance.Name))
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, *NewSentinelContainer(instance))
	return sts
}

func NewReplicationStatefulSet(instance *kvrocksv1alpha1.KVRocks) *kruise.StatefulSet {
	sts := NewStatefulSet(instance, GetStatefulSetName(instance.Name))
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, *NewInstanceContainer(instance), *NewExporterContainer(instance))
	return sts
}

func NewClusterStatefulSet(instance *kvrocksv1alpha1.KVRocks, index int) *kruise.StatefulSet {
	sts := NewStatefulSet(instance, GetStatefulSetName(instance.Name, index))
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, *NewInstanceContainer(instance), *NewExporterContainer(instance))
	return sts
}

func GetPVCOrPodIndex(podName string) (int, error) {
	index := podName[strings.LastIndex(podName, "-")+1:]
	return strconv.Atoi(index)
}

func GetStatefulSetName(name string, index ...int) string {
	if len(index) != 0 {
		return fmt.Sprintf("%s-%d", name, index[0])
	}
	return name
}

// default storage for controller
func NewEtcdStatefulSet(instance *kvrocksv1alpha1.KVRocks) *appsv1.StatefulSet {
	replicas := int32(1)

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvrocks.EtcdStatefulName,
			Namespace: instance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "etcd"},
			},
			ServiceName: kvrocks.EtcdServiceName,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "etcd"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "etcd",
							Image: "quay.io/coreos/etcd:latest",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: kvrocks.EtcdServerPort,
								}, {
									ContainerPort: kvrocks.EtcdClientPort,
								},
							},
							Args: []string{
								"/usr/local/bin/etcd",
								"--name=etcd0",
								"--listen-peer-urls=http://0.0.0.0:" + strconv.Itoa(kvrocks.EtcdServerPort),
								"--listen-client-urls=http://0.0.0.0:" + strconv.Itoa(kvrocks.EtcdClientPort),
								"--advertise-client-urls=http://" + kvrocks.EtcdServiceName + ":" + strconv.Itoa(kvrocks.EtcdClientPort),
								"--initial-advertise-peer-urls=http://" + kvrocks.EtcdServiceName + ":" + strconv.Itoa(kvrocks.EtcdServerPort),
								"--initial-cluster=etcd0=http://" + kvrocks.EtcdServiceName + ":" + strconv.Itoa(kvrocks.EtcdServerPort),
								"--initial-cluster-state=new",
							},
						},
					},
				},
			},
		},
	}
}
