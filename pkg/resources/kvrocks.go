package resources

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	uuid "github.com/satori/go.uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
)

// redis image

const SentinelImage = "redis:6.2.4"

var (
	ErrorPasswordEmpty        = "passworUnreasonable"
	ErrorReplicasUnreasonable = "replicasUnreasonable"
	ErrorResourcesNULL        = "resourcesUnreasonable"
)

func ValidateKVRocks(instance *kvrocksv1alpha1.KVRocks, log logr.Logger) (bool, *string) {
	if instance.Spec.Password == "" {
		log.Error(errors.New(ErrorPasswordEmpty), "password can not be blank")
		return false, &ErrorPasswordEmpty
	}
	if instance.Spec.Resources == nil {
		log.Error(errors.New(ErrorResourcesNULL), "resources must be not empty")
		return false, &ErrorResourcesNULL
	}
	switch instance.Spec.Type {
	case kvrocksv1alpha1.SentinelType:
		if instance.Spec.Replicas < 2 || instance.Spec.Replicas%2 == 0 {
			log.Error(errors.New(ErrorReplicasUnreasonable), "replicas must be greater than 2 in sentinel mode, and must be odd")
			return false, &ErrorReplicasUnreasonable
		}
	case kvrocksv1alpha1.StandardType:
		if instance.Spec.Master != 1 {
			log.Error(errors.New(ErrorReplicasUnreasonable), "master must be equal 1 in standard mode")
			return false, &ErrorReplicasUnreasonable
		}
	case kvrocksv1alpha1.ClusterType:
		if instance.Spec.Master < 3 {
			log.Error(errors.New(ErrorReplicasUnreasonable), "master must be greater than 3")
			return false, &ErrorReplicasUnreasonable
		}
	}
	return true, nil
}

func ParseRedisName(name string) (string, string) {
	// kvrocks-standard-22-test
	// kvrocks-cluster-22-test
	fields := strings.SplitN(name, "-", 4)
	if len(fields) < 4 {
		return "", ""
	}
	// sysId, name
	return fields[2], fields[3]
}

func GetSentinelInstance(instance *kvrocksv1alpha1.KVRocks) *kvrocksv1alpha1.KVRocks {
	system, _ := ParseRedisName(instance.Name)
	return &kvrocksv1alpha1.KVRocks{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSentinelName(system),
			Namespace: instance.Namespace,
		},
		Spec: kvrocksv1alpha1.KVRocksSpec{
			Image:           SentinelImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Type:            kvrocksv1alpha1.SentinelType,
			KVRocksConfig:   nil,
			Replicas:        3,
			Password: func(password string) string {
				d := []byte(password)
				m := md5.New()
				m.Write(d)
				return hex.EncodeToString(m.Sum(nil))
			}(system),
			Resources: &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("10m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("10m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
			},
			NodeSelector: instance.Spec.NodeSelector,
			Toleration:   instance.Spec.Toleration,
			Affinity:     instance.Spec.Affinity,
		},
	}
}

var key = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
	'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'}

func SetClusterNodeId() string {
	rand.Seed(time.Now().Unix())
	uid := uuid.NewV4().String()
	for i := 1; i <= 4; i++ {
		v1 := key[rand.Intn(len(key))]
		v2 := key[rand.Intn(len(key))]
		uid = strings.Replace(uid, "-", fmt.Sprintf("%c%c", v1, v2), 1)
	}
	rand.Intn(len(key))
	return uid
}

func GetClusterName(systenId, name string) string {
	return fmt.Sprintf("kvrocks-cluster-%s-%s", systenId, name)
}
