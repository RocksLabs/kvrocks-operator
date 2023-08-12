package resources

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

func NewSentinelContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	container := newSentinelContainer(instance)
	container.Command = []string{"sh", "-c", "cp -n /conf/sentinel.conf /data/sentinel.conf; redis-server /data/sentinel.conf --sentinel"}
	container.Ports = []corev1.ContainerPort{{
		Name:          "sentinel",
		ContainerPort: 26379,
	}}
	return container
}

func NewInstanceContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	container := newKVRocksContainer(instance)
	container.Command = []string{"sh", "/var/lib/kvrocks/conf/start.sh"}
	container.Ports = []corev1.ContainerPort{{
		Name:          "kvrocks",
		ContainerPort: kvrocks.KVRocksPort,
	}}
	return container
}

func NewExporterContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	return &corev1.Container{
		Name:  "kvrocks-exporter",
		Image: "hulkdev/kvrocks-exporter:latest",
		Args: []string{
			fmt.Sprintf("--kvrocks.addr=http://localhost:%s", strconv.Itoa(kvrocks.KVRocksPort)),
			fmt.Sprintf("--kvrocks.password=%s", instance.Spec.Password),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "exporter",
				ContainerPort: 9121,
			},
		},
	}
}

func newSentinelContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	cmd := []string{"redis-cli", "-p", "26379", "ping"}
	handler := corev1.ProbeHandler{
		Exec: &corev1.ExecAction{
			Command: cmd,
		},
	}
	container := &corev1.Container{
		Name:            "sentinel",
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

func newKVRocksContainer(instance *kvrocksv1alpha1.KVRocks) *corev1.Container {
	cmd := []string{"sh", "/var/lib/kvrocks/conf/readiness_probe.sh"}
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
				MountPath: "/var/lib/kvrocks",
			},
			{
				Name:      "conf",
				MountPath: "/var/lib/kvrocks/conf",
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
