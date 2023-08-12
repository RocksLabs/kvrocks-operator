package util

import (
	chaosmeshv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (env *KubernetesEnv) ScheduleInjectPodKill(target chaosmeshv1alpha1.PodSelectorSpec, schedule string, mode chaosmeshv1alpha1.SelectorMode) *Experiment {
	return env.schedulInjectPod(chaosmeshv1alpha1.PodSelector{
		Selector: target,
		Mode:     mode,
	}, schedule, chaosmeshv1alpha1.PodKillAction)
}

func (env *KubernetesEnv) schedulInjectPod(target chaosmeshv1alpha1.PodSelector, schedule string, action chaosmeshv1alpha1.PodChaosAction) *Experiment {
	return env.CreateExperiment(&chaosmeshv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduleinjectpod",
			Namespace: ChaosMeshNamespace,
		},
		Spec: chaosmeshv1alpha1.ScheduleSpec{
			Schedule:          schedule,
			ConcurrencyPolicy: chaosmeshv1alpha1.AllowConcurrent,
			Type:              chaosmeshv1alpha1.ScheduleTypePodChaos,
			HistoryLimit:      1,
			ScheduleItem: chaosmeshv1alpha1.ScheduleItem{
				EmbedChaos: chaosmeshv1alpha1.EmbedChaos{
					PodChaos: &chaosmeshv1alpha1.PodChaosSpec{
						Action: action,
						ContainerSelector: chaosmeshv1alpha1.ContainerSelector{
							PodSelector: target,
						},
					},
				},
			},
		},
	})
}
