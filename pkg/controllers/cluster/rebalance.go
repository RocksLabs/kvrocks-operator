package cluster

import (
	"errors"
	"time"

	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

func (h *KVRocksClusterHandler) ensureMigrate() error {
	masters := make([]*kvrocks.Node, 0)
	h.masters = map[string]*kvrocks.Node{}
	for _, nodes := range h.stsNodes {
		for _, node := range nodes {
			if node.Role == kvrocks.RoleMaster {
				masters = append(masters, node)
				h.masters[node.NodeId] = node
			}
		}
	}
	for index, master := range masters {
		if master.Migrate != nil {
			h.requeue = true
			return h.ensureReBalanceTopo(index, master)
		}
	}
	h.log.Info("migrate successfully")
	return h.ensureStatusTopoMsg()
}

func (h *KVRocksClusterHandler) ensureReBalanceTopo(src int, node *kvrocks.Node) error {
	for _, migrate := range node.Migrate {
		dest := migrate.Shard
		h.log.Info("begin move slots", "src", src, "dst", dest, "slots", migrate.Slots)
		for _, slot := range migrate.Slots {
			retry := 0
			wait := time.Millisecond * 10
		moveSlots:
			err := h.controllerClient.MigrateSlotAndData(src, dest, slot)
			if err != nil {
				h.log.Error(err, "move slot error")
				if retry < 5 {
					time.Sleep(wait)
				} else {
					h.log.Error(errors.New("slot migrate timeout"), "slot migrate timeout")
					return errors.New("slot migrate timeout")
				}
				retry++
				wait *= 10
				goto moveSlots
			}
		}
		node.Slots = node.Slots[len(migrate.Slots):]
		node.Migrate = node.Migrate[1:]
		if err := h.ensureStatusTopoMsg(); err != nil {
			return err
		}
		h.log.Info("move slots successfully", "src", src, "dst", dest, "slots", migrate.Slots)
	}
	return nil
}
