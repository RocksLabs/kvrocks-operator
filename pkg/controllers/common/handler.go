package common

import (
	kvrocksv1alpha1 "github.com/KvrocksLabs/kvrocks-operator/api/v1alpha1"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/k8s"
	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

type CommandHandler struct {
	instance *kvrocksv1alpha1.KVRocks
	k8s      *k8s.Client
	kvrocks  *kvrocks.Client
	password string
}

func NewCommandHandler(instance *kvrocksv1alpha1.KVRocks, k8s *k8s.Client, kvrocks *kvrocks.Client, password string) *CommandHandler {
	return &CommandHandler{
		instance: instance,
		k8s:      k8s,
		kvrocks:  kvrocks,
		password: password,
	}
}

func (handler *CommandHandler) ChangePassword(password string) {
	handler.password = password
}
