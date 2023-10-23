package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

func NewSentinelService(instance *kvrocksv1alpha1.KVRocks) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    instance.Labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name: "sentinel",
				Port: kvrocks.SentinelPort,
			}},
			Selector: instance.Labels,
		},
	}
}

func NewKVRocksService(instance *kvrocksv1alpha1.KVRocks) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    instance.Labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "kvrocks",
					Port: kvrocks.KVRocksPort,
				},
			},
			Selector: MergeLabels(instance.Labels, map[string]string{
				KvrocksRole: kvrocks.RoleMaster,
			}),
		},
	}
}

func NewEtcdService(instance *kvrocksv1alpha1.KVRocks) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvrocks.EtcdServiceName,
			Namespace: instance.Namespace,
			Labels:    map[string]string{"app": "etcd"},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "client",
					Port:       kvrocks.EtcdClientPort,
					TargetPort: intstr.FromInt(kvrocks.EtcdClientPort),
				}, {
					Name:       "server",
					Port:       kvrocks.EtcdServerPort,
					TargetPort: intstr.FromInt(kvrocks.EtcdServerPort),
				},
			},
			Selector: map[string]string{"app": "etcd"},
		},
	}
}

func NewKVRocksControllerService(instance *kvrocksv1alpha1.KVRocks) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvrocks.ControllerServiceName,
			Namespace: instance.Namespace,
			Labels:    map[string]string{"app": "kvrocks-controller"},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "kvrocks-controller"},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       kvrocks.ControllerPort,
					TargetPort: intstr.FromInt(kvrocks.ControllerPort),
				},
			},
		},
	}
}
