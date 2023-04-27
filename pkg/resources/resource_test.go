package resources

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

func TestSetClusterNodeId(t *testing.T) {
	for i := 0; i < 20; i++ {
		str := SetClusterNodeId()
		fmt.Println(str)
		fmt.Println(len(str))
	}
}

func TestSetSlots(t *testing.T) {
	slotsPreNode := (kvrocks.HashSlotCount) / 11
	slotsRem := (kvrocks.HashSlotCount) % 11
	allocated := 0
	for index := 0; index < 11; index++ {
		begin := allocated
		expected := slotsPreNode
		if index < slotsRem {
			expected++
		}
		for i := 0; i < expected; i++ {
			allocated++
		}
		end := allocated - 1
		fmt.Printf("begin: %d   end: %d\n", begin, end)
	}
}

func TestGetSlotSum(t *testing.T) {
	a := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	sort.Ints(a)
	fmt.Println(a)

}

func TestSlotToString(t *testing.T) {
	fmt.Println(time.Since(time.Now().Add(time.Second*30)) < 0)
}
