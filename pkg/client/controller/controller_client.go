package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RocksLabs/kvrocks-operator/pkg/client/k8s"
	"github.com/RocksLabs/kvrocks-operator/pkg/client/kvrocks"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type ClusterOption struct {
	Name     string   `json:"name"`
	Nodes    []string `json:"nodes"`
	Replicas int      `json:"replicas"`
	Password string   `json:"password"`
}

type NodeOption struct {
	Addr     string `json:"addr"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type MigrationOption struct {
	Source int `json:"source"`
	Target int `json:"target"`
	Slot   int `json:"slot"`
}

type Node struct {
	ID        string `json:"id"`
	Addr      string `json:"addr"`
	Role      string `json:"role"`
	Password  string `json:"password"`
	CreatedAt int64  `json:"created_at"`
}

type ShardData struct {
	Nodes         []Node   `json:"nodes"`
	SlotRanges    []string `json:"slot_ranges"`
	ImportSlot    int      `json:"import_slot"`
	MigratingSlot int      `json:"migrating_slot"`
}

type ShardOption struct {
	Nodes    []string `json:"nodes"`
	Password string   `json:"password"`
}

type Controller struct {
	EndPoint    string
	Namespace   string
	ClusterName string
}

type Client struct {
	logger     logr.Logger
	client     *http.Client
	controller *Controller
}

func NewClient(logger logr.Logger) *Client {
	return &Client{
		logger: logger,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		controller: &Controller{
			Namespace:   "cluster-demo",
			ClusterName: "cluster-demo",
		},
	}
}

func (c *Client) SetEndPoint(namespace string, k8s *k8s.Client) error {
	service, err := k8s.GetService(types.NamespacedName{
		Namespace: namespace,
		Name:      kvrocks.ControllerServiceName,
	})
	if err != nil {
		return err
	}
	c.controller.EndPoint = fmt.Sprintf("http://%s:%d/api/v1", service.Spec.ClusterIP, kvrocks.ControllerPort)
	return nil
}

func (c *Client) CreateIfNotExistsNamespace() error {
	resp, err := c.client.Post(c.controller.EndPoint+"/namespaces", "application/json", strings.NewReader(`{"namespace": "`+c.controller.Namespace+`"}`))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var errorResponse ErrorResponse
		err = json.Unmarshal(bodyBytes, &errorResponse)
		if err != nil {
			return err
		}
		if errorResponse.Error.Message != "the entry already existed" {
			return errors.New(errorResponse.Error.Message)
		} else {
			return nil
		}
	}
	return nil
}

func (c *Client) CreateCluster(replicas int, nodes []string, password string) error {
	clusterOption := &ClusterOption{
		Name:     c.controller.ClusterName,
		Replicas: replicas,
		Nodes:    nodes,
		Password: password,
	}
	clusterOptionJson, err := json.Marshal(clusterOption)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(c.controller.EndPoint+"/namespaces/"+c.controller.Namespace+"/clusters", "application/json", strings.NewReader(string(clusterOptionJson)))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return err
	}
	return nil
}

func (c *Client) CreateShard(nodes []string, password string) error {
	shardOption := &ShardOption{
		Nodes:    nodes,
		Password: password,
	}
	shardOptionJson, err := json.Marshal(shardOption)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(c.controller.EndPoint+"/namespaces/"+c.controller.Namespace+"/clusters/"+c.controller.ClusterName+"/shards", "application/json", strings.NewReader(string(shardOptionJson)))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return errors.New("unexpected response status code: " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func (c *Client) GetShards() ([]ShardData, error) {
	resp, err := c.client.Get(c.controller.EndPoint + "/namespaces/" + c.controller.Namespace + "/clusters/" + c.controller.ClusterName + "/shards")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseMap map[string]map[string][]ShardData
	err = json.Unmarshal(bodyBytes, &responseMap)
	if err != nil {
		return nil, err
	}

	shardData, ok := responseMap["data"]["shards"]
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	return shardData, nil
}

func (c *Client) DeleteShard(shardIndex int) error {
	req, err := http.NewRequest("DELETE", c.controller.EndPoint+"/namespaces/"+c.controller.Namespace+"/clusters/"+c.controller.ClusterName+"/shards/"+strconv.Itoa(shardIndex), nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected response status code: " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func (c *Client) GetNodes(shardIndex int) (*ShardData, error) {
	resp, err := c.client.Get(c.controller.EndPoint + "/namespaces/" + c.controller.Namespace + "/clusters/" + c.controller.ClusterName + "/shards/" + strconv.Itoa(shardIndex))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseMap map[string]map[string]ShardData
	err = json.Unmarshal(bodyBytes, &responseMap)
	if err != nil {
		return nil, err
	}

	shardData, ok := responseMap["data"]["shard"]
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	return &shardData, nil
}

func (c *Client) DeleteNode(shardIndex int, nodeID string) error {
	req, err := http.NewRequest("DELETE", c.controller.EndPoint+"/namespaces/"+c.controller.Namespace+"/clusters/"+c.controller.ClusterName+"/shards/"+strconv.Itoa(shardIndex)+"/nodes/"+nodeID, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected response status code: " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func (c *Client) AddNode(shardIndex int, addr, role, password string) error {
	nodeOption := &NodeOption{
		Addr:     addr,
		Role:     role,
		Password: password,
	}
	nodeOptionJson, err := json.Marshal(nodeOption)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(c.controller.EndPoint+"/namespaces/"+c.controller.Namespace+"/clusters/"+c.controller.ClusterName+"/shards/"+strconv.Itoa(shardIndex)+"/nodes", "application/json", strings.NewReader(string(nodeOptionJson)))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return errors.New("unexpected response status code: " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func (c *Client) FailoverShard(shardIndex int) error {
	resp, err := c.client.Post(c.controller.EndPoint+"/namespaces/"+c.controller.Namespace+"/clusters/"+c.controller.ClusterName+"/shards/"+strconv.Itoa(shardIndex)+"/failover", "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected response status code: " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func (c *Client) MigrateSlotAndData(source, target, slot int) error {
	migrationOption := &MigrationOption{
		Source: source,
		Target: target,
		Slot:   slot,
	}
	migrationOptionJson, err := json.Marshal(migrationOption)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(c.controller.EndPoint+"/namespaces/"+c.controller.Namespace+"/clusters/"+c.controller.ClusterName+"/shards/migration/slot_data", "application/json", strings.NewReader(string(migrationOptionJson)))

	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected response status code: " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}
