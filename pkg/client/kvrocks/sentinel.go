package kvrocks

import (
	"strconv"

	"github.com/go-redis/redis/v8"
)

func (s *Client) GetMasterFromSentinel(sentinelIP, sentinelPassword, master string) (string, error) {
	c := kvrocksSentinelClient(sentinelIP, sentinelPassword)
	defer c.Close()
	res, err := c.Master(ctx, master).Result()
	if err != nil {
		return "", err
	}
	masterIP := res["ip"]
	s.logger.V(1).WithValues("sentinel", sentinelIP, "master", master, "masterIP", masterIP).V(1).Info("get master from sentinel")
	return masterIP, nil
}

func (s *Client) RemoveMonitor(sentinelIP, password, master string) error {
	c := kvrocksSentinelClient(sentinelIP, password)
	defer c.Close()
	if err := c.Remove(ctx, master).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("sentinel remove master successfully", "master", master)
	return nil
}

func (s *Client) CreateMonitor(sentinelIP, password, master, ip, kvPass string) error {
	c := kvrocksSentinelClient(sentinelIP, password)
	defer c.Close()
	var err error
	if err = c.Monitor(ctx, master, ip, strconv.Itoa(KVRocksPort), strconv.Itoa(Quorum)).Err(); err != nil {
		return err
	}
	if err = c.Set(ctx, master, "AUTH-PASS", kvPass).Err(); err != nil {
		return err
	}
	if err = c.Set(ctx, master, "failover-timeout", "30000").Err(); err != nil {
		return err
	}
	if err = c.Set(ctx, master, "down-after-milliseconds", "10000").Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("sentinel monitor master successfully", "master", master)
	return nil
}

func (s *Client) ResetMonitor(sentinelIP, sentinelPassword, master, password string) error {
	c := kvrocksSentinelClient(sentinelIP, sentinelPassword)
	defer c.Close()
	var err error
	if err = c.Reset(ctx, master).Err(); err != nil {
		return err
	}
	if err = c.Set(ctx, master, "AUTH-PASS", password).Err(); err != nil {
		return err
	}
	s.logger.V(1).Info("sentinel reset master successfully", "master", master)
	return nil
}

func (s *Client) SubOdownMsg(ip, password string) (*redis.PubSub, func()) {
	c := kvrocksSentinelClient(ip, password)
	pubsub := c.Subscribe(ctx, "+odown")
	finalize := func() {
		pubsub.Close()
		c.Close()
	}

	return pubsub, finalize

}
