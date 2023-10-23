package events

import (
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/types"

	"github.com/RocksLabs/kvrocks-operator/pkg/client/controller"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/k8s"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
)

type eventMessage struct {
	ip        string
	port      string
	key       types.NamespacedName
	partition int
	timeout   time.Time
}

type messageQueue struct {
	message chan *eventMessage
	keys    map[string]struct{}
}

type produceMessage struct {
	ip       string
	password string
	key      string
	systemId string
}

var message = &messageQueue{
	message: make(chan *eventMessage, 1000),
	keys:    map[string]struct{}{},
}

type event struct {
	lock              sync.Mutex
	messages          *messageQueue
	producerSentinels map[string]func(msg *produceMessage)
	k8s               *k8s.Client
	kvrocks           *kvrocks.Client
	controller        *controller.Client
	log               logr.Logger
}

func NewEvent(k8s *k8s.Client, kvrocks *kvrocks.Client, controller *controller.Client, log logr.Logger) *event {
	return &event{
		k8s:               k8s,
		kvrocks:           kvrocks,
		controller:        controller,
		messages:          message,
		producerSentinels: map[string]func(msg *produceMessage){},
		log:               log,
	}
}

func (e *event) Run() {
	e.log.Info("begin listening failover messages")
	c := cron.New()
	c.AddFunc("@every 30s", e.producer)
	c.Start()
	e.consumer()
}

func (m *messageQueue) add(msg *eventMessage) {
	if _, ok := m.keys[msg.ip]; !ok {
		m.keys[msg.ip] = struct{}{}
		m.message <- msg
	}
}

func SendFailoverMsg(ip string, key types.NamespacedName, partition int) {
	message.add(&eventMessage{
		ip:        ip,
		port:      strconv.Itoa(kvrocks.KVRocksPort),
		key:       key,
		partition: partition,
		timeout:   time.Now().Add(time.Second * 30),
	})
}
