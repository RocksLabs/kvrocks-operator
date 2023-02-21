package kvrocks

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	client "github.com/go-redis/redis/v8"
)

func (s *Client) NodeInfo(ip, password string) (node Node, err error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	cmd := client.NewSliceCmd(ctx, "ROLE")
	c.Process(ctx, cmd)
	resp, err := cmd.Result()
	if err != nil || len(resp) == 0 {
		return
	}
	role := resp[0].(string)
	node.IP = ip
	node.Role = role
	return
}

func (s *Client) GetConfig(ip, password, key string) (*string, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	value, err := c.ConfigGet(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(value) != 2 {
		return nil, errors.New("invalid config key: " + key)
	}
	result := value[1].(string)
	return &result, nil
}

func (s *Client) SetConfig(ip, password string, key, value string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.ConfigSet(ctx, key, value).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("kvrocks set config successfully", "ip", ip, "key", key, "value", value)
	return nil
}

func (s *Client) ChangePassword(ip, password, newPassword string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	pipe := c.Pipeline()
	pipe.ConfigSet(ctx, "masterauth", newPassword)
	pipe.ConfigSet(ctx, "requirepass", newPassword)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	s.logger.V(1).Info("password changed successfully", "ip", ip)
	return nil
}

func (s *Client) ChangeMyselfToMaster(ip, password string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.SlaveOf(ctx, "NO", "ONE").Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("change myself to master successfully")
	return nil
}

func (s *Client) GetMaster(ip, password string) (string, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	info, err := c.Info(ctx, "replication").Result()
	if err != nil {
		return "", err
	}
	match := regexp.MustCompile("master_host:([0-9a-zA-Z\\.]+)").FindStringSubmatch(info)
	if len(match) == 0 {
		return "", nil
	}
	master := match[1]
	s.logger.V(1).Info("get master", "masterIP", master, "slaveIP", ip)
	return master, nil
}

func (s *Client) SlaveOf(slaveIP, masterIP, password string) error {
	c := kvrocksClient(slaveIP, password)
	defer c.Close()
	if err := c.SlaveOf(ctx, masterIP, strconv.Itoa(KVRocksPort)).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("slave of start", "master", masterIP, "slave", slaveIP)
	return nil
}

func (s *Client) GetOffset(ip, password string) (int, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	msg, err := c.Info(ctx, "replication").Result()
	if err != nil {
		return -1, err
	}
	lines := strings.Split(msg, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "slave_repl_offset") {
			offset, _ := strconv.Atoi(strings.Split(line, ":")[1])
			return offset, nil
		}
	}
	return -1, nil
}

func (s *Client) Ping(ip, password string) bool {
	c := kvrocksClient(ip, password)
	defer c.Close()
	timeout, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	if err := c.Ping(timeout).Err(); err != nil {
		return false
	}
	return true
}
