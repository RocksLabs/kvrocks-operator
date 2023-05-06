package resources

import (
	corev1 "k8s.io/api/core/v1"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

func NewSentinelContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	container := newKVRocksContainer(instance)
	container.Command = []string{"sh", "-c", "cp -n /conf/sentinel.conf /data/sentinel.conf; redis-server /data/sentinel.conf --sentinel"}
	container.Ports = []corev1.ContainerPort{{
		Name:          "sentinel",
		ContainerPort: 26379,
	}}
	return container
}

func NewInstanceContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	container := newKVRocksContainer(instance)
	container.Command = []string{"sh", "/conf/start.sh"}
	container.Ports = []corev1.ContainerPort{{
		Name:          "kvrocks",
		ContainerPort: kvrocks.KVRocksPort,
	}}
	return container
}

func newKVRocksContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	cmd := []string{"sh", "/conf/readiness_probe.sh"}
	if instance.Spec.Type == kvrocksv1alpha1.SentinelType {
		cmd = []string{"redis-cli", "-p", "26379", "ping"}
	}
	handler := corev1.ProbeHandler{
		Exec: &corev1.ExecAction{
			Command: cmd,
		},
	}
	container := &corev1.Container{
		Name:            "kvrocks",
		Image:           instance.Spec.Image,
		ImagePullPolicy: instance.Spec.ImagePullPolicy,
		Resources:       *instance.Spec.Resources,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data",
			},
			{
				Name:      "conf",
				MountPath: "/conf",
			},
		},
		ReadinessProbe: &corev1.Probe{
			TimeoutSeconds:   5,
			ProbeHandler:     handler,
			FailureThreshold: 6,
		},
		LivenessProbe: &corev1.Probe{
			TimeoutSeconds:   5,
			ProbeHandler:     handler,
			FailureThreshold: 6,
		},
	}
	return container
}
