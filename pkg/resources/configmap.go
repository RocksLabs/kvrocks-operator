package resources

import (
	"bytes"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
)

var UnChangeCfg = map[string]struct{}{
	"daemonize":                             {},
	"bind":                                  {},
	"port":                                  {},
	"workers":                               {},
	"tcp-backlog":                           {},
	"slaveof":                               {},
	"db-name":                               {},
	"dir":                                   {},
	"backup-dir":                            {},
	"log-dir":                               {},
	"pidfile":                               {},
	"supervised":                            {},
	"rename-command":                        {},
	"rocksdb.block_size":                    {},
	"rocksdb.wal_size_limit_mb":             {},
	"rocksdb.enable_pipelined_write":        {},
	"rocksdb.cache_index_and_filter_blocks": {},
	"rocksdb.subkey_block_cache_size":       {},
	"rocksdb.metadata_block_cache_size":     {},
	"rocksdb.share_metadata_and_subkey_block_cache": {},
	"rocksdb.row_cache_size":                        {},
	"masterauth":                                    {},
	"requirepass":                                   {},
	"cluster-enabled":                               {},
}

const (
	// operator -> kvrocks/sentinel
	superUser = "user superuser ~* +@all on >%s\n"
	// client -> sentinel
	sentinelDefaultUser = "user default +@all -acl -sentinel +sentinel|master +sentinel|replicas +sentinel|sentinels +sentinel|get-master-addr-by-name +sentinel|is-master-down-by-addr +sentinel|slaves on nopass\n"
	// sentinel -> kvrocks
	sentinelUser = "user sentinel allchannels +multi +slaveof +ping +exec +subscribe +config|rewrite +role +publish +info +client|setname +client|kill +script|kill on >%s\n"
)

const (
	start = `
#!/bin/bash
sleep 15
./bin/kvrocks -c /conf/kvrocks.conf
`

	readinessProbe = `
#!/bin/sh
timeout 30 redis-cli -a $(cat /conf/password) --no-auth-warning ping || timeout 30 redis-cli --no-auth-warning ping
`
)

func NewSentinelConfigMap(instance *kvrocksv1alpha1.KVRocks) *corev1.ConfigMap {
	var buffer bytes.Buffer
	buffer.WriteString(sentinelDefaultUser)
	buffer.WriteString(fmt.Sprintf(superUser, instance.Spec.Password))
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
			Labels: instance.Labels,
		},
		Data: map[string]string{
			"sentinel.conf": buffer.String(),
		},
	}
	return configMap
}

func NewKVRocksConfigMap(instance *kvrocksv1alpha1.KVRocks) *corev1.ConfigMap {
	var buffer bytes.Buffer
	// add kvrocks config
	for k, v := range instance.Spec.KVRocksConfig {
		if v == "" {
			buffer.WriteString(fmt.Sprintf("%s \"\"\n", k))
		} else {
			buffer.WriteString(fmt.Sprintf("%s %s\n", k, v))
		}
	}
	// set auth config
	buffer.WriteString(fmt.Sprintf("masterauth %s\n", instance.Spec.Password))
	buffer.WriteString(fmt.Sprintf("requirepass %s\n", instance.Spec.Password))
	buffer.WriteString("dir /data\n")
	if instance.Spec.Type == kvrocksv1alpha1.ClusterType {
		buffer.WriteString("cluster-enabled yes\n")
	}
	if instance.Spec.Type == kvrocksv1alpha1.StandardType {
		buffer.WriteString("slaveof 127.0.0.1 6379\n")
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, instance.GroupVersionKind()),
			},
			Labels: instance.Labels,
		},
		Data: map[string]string{
			"kvrocks.conf":       buffer.String(),
			"password":           instance.Spec.Password,
			"start.sh":           start,
			"readiness_probe.sh": readinessProbe,
		},
	}
}

func ParseKVRocksConfigs(config map[string]string) map[string]string {
	cfg := make(map[string]string)
	for key, value := range config {
		if _, ok := UnChangeCfg[key]; ok {
			continue
		}
		cfg[key] = value
	}
	return cfg
}
