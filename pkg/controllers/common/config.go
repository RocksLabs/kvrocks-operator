package common

import (
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *CommandHandler) EnsureConfig(nodes []*kvrocks.Node) error {
	config := resources.ParseKVRocksConfigs(h.instance.Spec.KVRocksConfig)
	for _, node := range nodes {
		for key, value := range config {
			curValue, err := h.kvrocks.GetConfig(node.IP, h.password, key)
			if err != nil {
				return err
			}
			if *curValue != value {
				if err = h.kvrocks.SetConfig(node.IP, h.password, key, value); err != nil {
					return err
				}
			}
		}
		if h.password != h.instance.Spec.Password {
			if err := h.kvrocks.ChangePassword(node.IP, h.password, h.instance.Spec.Password); err != nil {
				return err
			}
		}
	}
	return nil
}
