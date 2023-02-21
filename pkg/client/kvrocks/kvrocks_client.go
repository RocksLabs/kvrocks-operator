package kvrocks

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	client "github.com/go-redis/redis/v8"
)

var ctx = context.TODO()

const (
	KVRocksPort   = 6379
	SentinelPort  = 26379
	SuperUser     = "superuser"
	RoleMaster    = "master"
	RoleSlaver    = "slave"
	Quorum        = 2
	HashSlotCount = 16384
)

const ErrPassword = "ERR invalid password"

type Node struct {
	IP       string
	Role     string
	PodIndex int
	Slots    []int
	NodeId   string
	Master   string
	Expected int
	Failover bool
	Migrate  []MigrateMsg
	Import   []ImportMsg
}

type ImportMsg struct {
	SrcNodeId string
	Slots     []int
}

type MigrateMsg struct {
	DstNodeID string
	Slots     []int
}

type Client struct {
	logger logr.Logger
}

func NewKVRocksClient(logger logr.Logger) *Client {
	return &Client{logger: logger}
}

func kvrocksClient(ip, password string) *client.Client {
	return client.NewClient(&client.Options{
		Addr:     net.JoinHostPort(ip, strconv.Itoa(KVRocksPort)),
		Password: password,
	})
}

func kvrocksSentinelClient(ip, password string) *client.SentinelClient {
	return client.NewSentinelClient(&client.Options{
		Addr:     net.JoinHostPort(ip, strconv.Itoa(SentinelPort)),
		Username: SuperUser,
		Password: password,
	})
}

func kvrocksClusterClient(ip, password string) *client.ClusterClient {
	return client.NewClusterClient(&client.ClusterOptions{
		Addrs:    []string{net.JoinHostPort(ip, strconv.Itoa(KVRocksPort))},
		Password: password,
	})
}

func (node *Node) InsertSlot(value int) {
	node.Slots = append(node.Slots, value)
	sort.Ints(node.Slots)
}

func SlotsToString(slots []int) []string {
	sort.Ints(slots)
	l := len(slots)
	var result []string
	if l == 0 {
		return result
	}
	head := slots[0]
	for i := 1; i < l; i++ {
		if slots[i]-slots[i-1] != 1 {
			if head != slots[i-1] {
				result = append(result, fmt.Sprintf("%d-%d", head, slots[i-1]))
			} else {
				result = append(result, fmt.Sprintf("%d", head))
			}
			head = slots[i]
		}
	}
	if head == slots[l-1] {
		result = append(result, fmt.Sprintf("%d", head))
	} else {
		result = append(result, fmt.Sprintf("%d-%d", head, slots[l-1]))
	}
	return result
}

func SlotsToInt(slots []string) []int {
	var result []int
	for _, slot := range slots {
		field := strings.Split(slot, "-")
		if len(field) == 1 {
			slotNum, _ := strconv.Atoi(field[0])
			result = append(result, slotNum)
		} else {
			begin, _ := strconv.Atoi(field[0])
			end, _ := strconv.Atoi(field[1])
			for i := begin; i <= end; i++ {
				result = append(result, i)
			}
		}
	}
	return result
}
