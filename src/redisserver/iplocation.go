package redisserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SongZihuan/ssh-watcher/src/api/apiip"
	"github.com/SongZihuan/ssh-watcher/src/logger"
	"net"
	"time"
)

const IspLoopback = "本地回环地址"
const IspIntranet = "内网地址"

func QueryNetIpLocation(ipNet net.IP) (*apiip.QueryIpLocationData, error) {
	if ipNet.IsLoopback() {
		return &apiip.QueryIpLocationData{Isp: IspLoopback}, nil
	} else if ipNet.IsPrivate() {
		return &apiip.QueryIpLocationData{Isp: IspIntranet}, nil
	}

	return queryIpLocation(ipNet.String())
}

func QueryIpLocation(ip string) (*apiip.QueryIpLocationData, error) {
	ipNet := net.ParseIP(ip)
	if ipNet == nil {
		return nil, fmt.Errorf("ip is not valid: %s", ip)
	}

	if ipNet.IsLoopback() {
		return &apiip.QueryIpLocationData{Isp: IspLoopback}, nil
	} else if ipNet.IsPrivate() {
		return &apiip.QueryIpLocationData{Isp: IspIntranet}, nil
	}

	return queryIpLocation(ip)
}

func queryIpLocation(ip string) (*apiip.QueryIpLocationData, error) {
	key := fmt.Sprintf("ip:location:%s", ip)

	cacheRes := func() *apiip.QueryIpLocationData {
		res, err := rdb.Get(context.Background(), key).Result()
		if err != nil {
			return nil
		}

		var loc apiip.QueryIpLocationData
		err = json.Unmarshal([]byte(res), &loc)
		if err != nil {
			return nil
		}

		return &loc
	}()
	if cacheRes != nil {
		return cacheRes, nil
	}

	res, err := apiip.QueryIpLocation(ip)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	err = rdb.Set(context.Background(), key, string(data), 24*time.Hour).Err()
	if err != nil {
		logger.Errorf("redis set error: %s", err.Error())
	}

	return res, nil
}
