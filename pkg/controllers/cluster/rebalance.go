package cluster

import (
	"errors"
	"time"

	"github.com/KvrocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

func (h *KVRocksClusterHandler) reBalance() error {
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
	for _, master := range masters {
		if master.Migrate != nil {
			h.requeue = true
			return h.ensureReBalanceTopo(master)
		}
	}
	h.instance.Status.Rebalance = false
	h.calcExpectSlots(masters)
	first := 0
	last := len(masters) - 1
	h.log.Info("begin reBalance!")
	for first < last {
		node1 := masters[first]
		node2 := masters[last]
		curLenNode1 := getCurSlotLen(node1)
		curLenNode2 := getCurSlotLen(node2)
		if node1.Expected == curLenNode1 {
			first++
			continue
		}
		if node2.Expected == curLenNode2 {
			last--
			continue
		}
		h.instance.Status.Rebalance = true
		moved, err := h.moveSlot(node1, node2)
		if err != nil {
			return err
		}
		if moved == 0 {
			last--
		}
	}
	h.log.Info("reBalance successfully")
	return h.ensureStatusTopoMsg()
}

func (h *KVRocksClusterHandler) ensureReBalanceTopo(node *kvrocks.Node) error {
	for _, migrate := range node.Migrate {
		dstNodeID := migrate.DstNodeID
		h.log.Info("begin move slots", "src", node.NodeId, "dst", dstNodeID, "slots", migrate.Slots)
		for _, slot := range migrate.Slots {
			retry := 0
			wait := time.Millisecond * 10
		moveSlots:
			if !h.kvrocks.MoveSlots(node.IP, h.password, slot, dstNodeID) {
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
			if err := h.restSlot(slot, dstNodeID); err != nil {
				return err
			}
		}
		node.Slots = node.Slots[len(migrate.Slots):]
		dst := h.masters[dstNodeID]
		dst.Slots = append(dst.Slots, migrate.Slots...)
		node.Migrate = node.Migrate[1:]
		dst.Import = dst.Import[1:]
		if err := h.ensureStatusTopoMsg(); err != nil {
			return err
		}
		h.log.Info("move slots successfully", "src", node.NodeId, "dst", dstNodeID, "slots", migrate.Slots)
	}
	return nil
}

func (h *KVRocksClusterHandler) restSlot(slot int, dstNodeId string) error {
	// h.version++
	for _, sts := range h.stsNodes {
		for _, node := range sts {
		reset:
			if err := h.kvrocks.ResetSlot(node.IP, h.password, slot, h.version, dstNodeId); err != nil {
				if err.Error() == kvrocks.ClusterVersionInvalid {
					h.version++
					goto reset
				}
				return err
			}
		}
	}
	return nil
}

func (h *KVRocksClusterHandler) moveSlot(node1 *kvrocks.Node, node2 *kvrocks.Node) (int, error) {
	balance1 := getCurSlotLen(node1) - node1.Expected
	balance2 := getCurSlotLen(node2) - node2.Expected
	moved := 0
	src, dst := node1, node2
	if balance1 > 0 && balance2 < 0 {
		moved = min(balance1, -balance2)
	}
	if balance1 < 0 && balance2 > 0 {
		moved = min(-balance1, balance2)
		src, dst = node2, node1
	}
	var slots []int
	index := 0
	for _, migrate := range src.Migrate {
		index += len(migrate.Slots)
	}
	for moved > 0 {
		slot := src.Slots[index]
		moved--
		index++
		slots = append(slots, slot)
	}
	if len(slots) != 0 {
		src.Migrate = append(src.Migrate, kvrocks.MigrateMsg{
			DstNodeID: dst.NodeId,
			Slots:     slots,
		})
		dst.Import = append(dst.Import, kvrocks.ImportMsg{
			SrcNodeId: src.NodeId,
			Slots:     slots,
		})
	}
	return len(slots), nil
}

func (h *KVRocksClusterHandler) calcExpectSlots(masters []*kvrocks.Node) {
	slotsPreNode := kvrocks.HashSlotCount / h.instance.Spec.Master
	slotsRem := kvrocks.HashSlotCount % h.instance.Spec.Master
	for i, node := range masters {
		node.Expected = int(slotsPreNode)
		if i < int(slotsRem) {
			node.Expected++
		}
		if i >= int(h.instance.Spec.Master) {
			node.Expected = 0
		}
	}
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func getCurSlotLen(node *kvrocks.Node) int {
	result := len(node.Slots)
	if node.Import != nil {
		for _, im := range node.Import {
			result += len(im.Slots)
		}
	}
	if node.Migrate != nil {
		for _, migrate := range node.Migrate {
			result -= len(migrate.Slots)
		}
	}
	return result
}
