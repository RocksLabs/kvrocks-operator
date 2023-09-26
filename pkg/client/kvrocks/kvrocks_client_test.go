package kvrocks

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/joaojeronimo/go-crc16"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var c Client

var sentinels = []string{"10.0.77.34", "10.0.76.245", "10.0.78.80"}

func init() {
	opts := zap.Options{
		Development: true,
	}
	// opts.BindFlags(flag.CommandLine)
	// flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	c = NewKVRocksClient(ctrl.Log)
}

func TestClient_NodeInfo(t *testing.T) {
	c.NodeInfo("10.0.77.78", "39c5bb")
}

func TestClient_GetMasterFromSentinel(t *testing.T) {
	master := "demo"
	for _, ip := range sentinels {
		for i := 0; i < 4; i++ {
			ip, err := c.GetMasterFromSentinel(ip, "c4ca4238a0b923820dcc509a6f75849b", fmt.Sprintf("%s-%d", master, i))
			if err != nil {
				panic(err.Error())
			}
			fmt.Println(ip)
		}
	}
}

func TestClient_RemoveMonitor(t *testing.T) {
	for _, ip := range sentinels {
		c.RemoveMonitor(ip, "c4ca4238a0b923820dcc509a6f75849b", "kvrocks")
	}
}

func TestClient_CreateMonitor(t *testing.T) {
	for _, ip := range sentinels {
		c.CreateMonitor(ip, "c4ca4238a0b923820dcc509a6f75849b", "kvrocks", "10.0.77.78", "123456")
	}
}

func TestClient_GetMaster(t *testing.T) {
	c.GetMaster("10.0.77.88", "39c5bb")
}

func TestSetClusterID(t *testing.T) {
	masterIp := "10.0.77.90"
	slaveIp := "10.0.77.100"
	if err := c.SetClusterID(masterIp, "39c5bb", "f7149f2apw3d8ftm4a01w59b35v75b46e8555ae4"); err != nil {
		c.Logger().Error(err, "set nodeID error")
	}
	if err := c.SetClusterID(slaveIp, "39c5bb", "2beb1a909fa2a8w54588i2bd51lo5936dfdd8ae8"); err != nil {
		c.Logger().Error(err, "set nodeID error")
	}
}

func TestClient_SetTopoMsg(t *testing.T) {
	masterNode := "f7149f2apw3d8ftm4a01w59b35v75b46e8555ae4"
	slaveNode := "2beb1a909fa2a8w54588i2bd51lo5936dfdd8ae8"
	masterIp := "10.0.77.90"
	slaveIp := "10.0.77.100"
	topoMsg := fmt.Sprintf("%s %s %d master - %d-%d\n%s %s %d slave %s", masterNode, masterIp, KVRocksPort, 0, 4096, slaveNode, slaveIp, KVRocksPort, masterNode)
	fmt.Println(len(masterNode))
	fmt.Println(topoMsg)
	if err := c.SetTopoMsg(slaveIp, "39c5bb", topoMsg, 3); err != nil {
		fmt.Println(err)
	}
	// fmt.Println( "2666fa2ce6db5dwv406e7ebd04gh4b2f2f439467 10.0.78.163 6379 master - 0-5462\n777e403be6af99wv41d37ea476gh00a055ce4649 10.0.77.8 6379 slave 2666fa2ce6db5dwv406e7ebd04gh4b2f2f439467\nd819b1d7e6072dwv40c87e887egh2ece4c412574 10.0.76.229 6379 master - 5461-10922\n7c385870e66a5ewv47d47ebd8agh05768425f5b2 10.0.76.180 6379 slave d819b1d7e6072dwv40c87e887egh2ece4c412574\n28873b1ee61484wv47107e82b1ghb6a73b339a86 10.0.78.78 6379 master - 10922-16383\n17a257b1e61514wv4ea67eb9bdghd3c033ba5164 10.0.77.19 6379 slave 28873b1ee61484wv47107e82b1ghb6a73b339a86")
}

func TestClient_Ping(t *testing.T) {
	ip := "10.0.76.148"
	fmt.Println(c.Ping(ip, "39c5bb"))
	var lock sync.RWMutex
	lock.Lock()
}

func TestClient_GetOffset(t *testing.T) {
	ip := "10.0.76.143"
	off, err := c.GetOffset(ip, "39c5bb")
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(off)
}

func Test_crc16(t *testing.T) {
	c := kvrocksClusterClient("10.0.78.140", "123456")
	defer c.Close()
	index := 0
	for {
		if index == 10000 {
			break
		}
		id := SetClusterNodeId()
		crc := crc16.Crc16([]byte(id))
		if crc%16384 < 5461 {
			index++
			c.Set(ctx, id, id, 0)
		}
	}
	fmt.Println("ok")
}

var key = []byte{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
	'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
}

func SetClusterNodeId() string {
	rand.Seed(time.Now().Unix())
	uid := uuid.New().String()
	for i := 1; i <= 4; i++ {
		v1 := key[rand.Intn(len(key))]
		v2 := key[rand.Intn(len(key))]
		uid = strings.Replace(uid, "-", fmt.Sprintf("%c%c", v1, v2), 1)
	}
	rand.Intn(len(key))
	return uid
}

func Test_DBSize(t *testing.T) {
	ips := []string{"10.0.76.162", "10.0.77.38", "10.0.78.115"}
	var sum int64 = 0
	for _, ip := range ips {
		c := kvrocksClient(ip, "123456")
		defer c.Close()
		c.Do(ctx, "dbsize", "scan")
		time.Sleep(time.Second * 2)
		result, err := c.DBSize(ctx).Result()
		if err != nil {
			panic(err.Error())
		}
		sum += result
		fmt.Println(result)
	}
	fmt.Println("sum: ", sum)
}

func Test_DeepEqual(t *testing.T) {
	ips := []string{"10.0.76.200", "10.0.76.162", "10.0.78.140", "10.0.77.38", "10.0.78.115", "10.0.76.170", "10.0.78.128", "10.0.78.83", "10.0.77.35", "10.0.76.189"}
	infos := map[string]string{}
	for _, ip := range ips {
		c := kvrocksClient(ip, "123456")
		defer c.Close()
		info, err := c.ClusterNodes(ctx).Result()
		infos[strings.ReplaceAll(info, "myself,", "")] = ip
		if err != nil {
			panic(err.Error())
		}
	}
	basic := ""
	index := 1
	for info, ip := range infos {
		if index == 1 {
			basic = info
			index++
			continue
		}
		if basic != info {
			fmt.Println("cluster nodes not equal")
			fmt.Println(ip)
			fmt.Println(infos[basic])
		}
	}
	fmt.Println("equal")
}

func TestClient_ClusterNodeInfo(t *testing.T) {
	node, err := c.ClusterNodeInfo("10.0.76.251", "123456")
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(SlotsToString(node.Slots))
	node.Slots = nil
	fmt.Printf("%+v\n", *node)
}

func TestClient_ClusterVersion(t *testing.T) {
	version, err := c.ClusterVersion("10.0.77.36", "123456")
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(version)
}

func TestClient_SetKey(t *testing.T) {
	c := kvrocksClusterClient("10.0.69.18", "39c5bb")
	defer c.Close()
	pipe := c.Pipeline()
	index := 0
	for i := 0; i < 1000000; i++ {
		index++
		for j := 0; j < 1000; j++ {
			pipe.Set(ctx, fmt.Sprintf("key-%d-%d", i, j), fmt.Sprintf("value-%d-%d", i, j), 0)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			panic(err.Error())
		}
		if index%100000 == 0 {
			fmt.Println("执行了 100000 次")
		}
	}
	fmt.Println("ok")
}
