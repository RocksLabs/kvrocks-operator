package kvrocks

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	redisClient "github.com/go-redis/redis/v8"
)

var ctx = context.TODO()

const (
	KVRocksPort  = 6379
	SentinelPort = 26379
	SuperUser    = "superuser"
	RoleMaster   = "master"
	RoleSlaver   = "slave"
	Quorum       = 2
	MinSlotID    = 0
	MaxSlotID    = 16383

	EtcdStatefulName = "etcd0"
	EtcdServiceName  = "etcd0-service"
	EtcdClientPort   = 2379
	EtcdServerPort   = 2380

	ControllerServiceName    = "controller-service"
	ControllerPort           = 9379
	ControllerDeploymentName = "kvrocks-controller"
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
}

type MigrateMsg struct {
	Shard int
	Slots []int
}

type client struct {
	logger logr.Logger
}

func (s *client) Logger() logr.Logger {
	return s.logger
}

type Client interface {
	Logger() logr.Logger

	ChangeMyselfToMaster(ip string, password string) error
	ChangePassword(ip string, password string, newPassword string) error
	ClusterNodeInfo(ip string, password string) (*Node, error)
	CreateMonitor(sentinelIP string, password string, master string, ip string, kvPass string) error
	GetConfig(ip string, password string, key string) (*string, error)
	GetMaster(ip string, password string) (string, error)
	GetMasterFromSentinel(sentinelIP string, sentinelPassword string, master string) (string, error)
	GetOffset(ip string, password string) (int, error)
	NodeInfo(ip string, password string) (node Node, err error)
	Ping(ip string, password string) bool
	RemoveMonitor(sentinelIP string, password string, master string) error
	ResetMonitor(sentinelIP string, sentinelPassword string, master string, password string) error
	SetConfig(ip string, password string, key string, value string) error
	SlaveOf(slaveIP string, masterIP string, password string) error
	SubOdownMsg(ip string, password string) (*redisClient.PubSub, func())
}

func NewKVRocksClient(logger logr.Logger) Client {
	return &client{logger: logger}
}

func kvrocksClient(ip, password string) *redisClient.Client {
	return redisClient.NewClient(&redisClient.Options{
		Addr:     net.JoinHostPort(ip, strconv.Itoa(KVRocksPort)),
		Password: password,
	})
}

func kvrocksSentinelClient(ip, password string) *redisClient.SentinelClient {
	return redisClient.NewSentinelClient(&redisClient.Options{
		Addr:     net.JoinHostPort(ip, strconv.Itoa(SentinelPort)),
		Username: SuperUser,
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
