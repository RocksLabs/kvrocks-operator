package resources

import (
	"fmt"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewSentinelDeployment(instance *kvrocksv1alpha1.KVRocks) *appsv1.Deployment {
	name := GetDeploymentName(instance.Name)
	labels := MergeLabels(instance.Labels, DeploymentLabels(name))
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Affinity:                      getAffinity(instance, labels),
					NodeSelector:                  instance.Spec.NodeSelector,
					Tolerations:                   instance.Spec.Toleration,
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
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

	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, getSentinelDataVolume(instance))

	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *NewSentinelContainer(instance))

	return dep
}

func getSentinelDataVolume(instance *kvrocksv1alpha1.KVRocks) corev1.Volume {
	return corev1.Volume{
		Name: "data",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func GetDeploymentName(name string, index ...int) string {
	if len(index) != 0 {
		return fmt.Sprintf("%s-%d", name, index[0])
	}
	return name
}
