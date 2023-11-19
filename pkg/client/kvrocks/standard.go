package kvrocks

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	redisClient "github.com/go-redis/redis/v8"
)

// NodeInfo returns the node info
func (s *client) NodeInfo(ip, password string) (node Node, err error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	cmd := redisClient.NewSliceCmd(ctx, "ROLE")
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

// GetConfig returns the config value
func (s *client) GetConfig(ip, password, key string) (*string, error) {
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

// SetConfig sets a single config in key value format
func (s *client) SetConfig(ip, password string, key, value string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.ConfigSet(ctx, key, value).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("kvrocks set config successfully", "ip", ip, "key", key, "value", value)
	return nil
}

// ChangePassword changes the password
func (s *client) ChangePassword(ip, password, newPassword string) error {
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

// ChangeMyselfToMaster changes the current node to master
func (s *client) ChangeMyselfToMaster(ip, password string) error {
	c := kvrocksClient(ip, password)
	defer c.Close()
	if err := c.SlaveOf(ctx, "NO", "ONE").Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("change myself to master successfully")
	return nil
}

// GetMaster returns the master ip
func (s *client) GetMaster(ip, password string) (string, error) {
	c := kvrocksClient(ip, password)
	defer c.Close()
	info, err := c.Info(ctx, "replication").Result()
	if err != nil {
		return "", err
	}
	match := regexp.MustCompile(`master_host:([0-9a-zA-Z\.]+)`).FindStringSubmatch(info)
	if len(match) == 0 {
		return "", nil
	}
	master := match[1]
	s.logger.V(1).Info("get master", "masterIP", master, "slaveIP", ip)
	return master, nil
}

// SlaveOf sets the slave of the specified master
func (s *client) SlaveOf(slaveIP, masterIP, password string) error {
	c := kvrocksClient(slaveIP, password)
	defer c.Close()
	if err := c.SlaveOf(ctx, masterIP, strconv.Itoa(KVRocksPort)).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("slave of start", "master", masterIP, "slave", slaveIP)
	return nil
}

// GetOffset returns the replication offset
func (s *client) GetOffset(ip, password string) (int, error) {
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

// Ping checks if the node is alive
func (s *client) Ping(ip, password string) bool {
	c := kvrocksClient(ip, password)
	defer c.Close()
	timeout, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	if err := c.Ping(timeout).Err(); err != nil {
		return false
	}
	return true
}
