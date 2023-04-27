package resources

import (
	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
)

const (
	MonitoredBy = "kvrocks/monitored-by"
	RedisRole   = "kvrocks/role"
)

func MergeLabels(allLabels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, labels := range allLabels {
		for k, v := range labels {
			res[k] = v
		}
	}
	return res
}

func StatefulSetLabels(name string) map[string]string {
	return map[string]string{
		"kvrocks/statefulset": name,
	}
}

func MonitorLabels(sentinel string) map[string]string {
	return map[string]string{
		MonitoredBy: sentinel,
	}
}

func SelectorLabels(instance *kvrocksv1alpha1.KVRocks) map[string]string {
	return map[string]string{
		"app.kubernetes.io/kind": instance.Kind,
		"kvrocks/name":           instance.Name,
	}
}

func SentinelLabels() map[string]string {
	return map[string]string{
		"sentinel": "true",
	}
}

func GetSentinelName(system string) string {
	return "sentinel-" + system
}
