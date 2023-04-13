package tool

import (
	"errors"
	"net"
)

var Ip = &ipCommon{}

type ipCommon struct {
}

// GetClientIp 获取本地ip地址
func (i *ipCommon) GetClientIp() (ip string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, addr := range addrs {
		// 检查IP地址是否为IPv4地址，如果是，则返回该地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ip = ipnet.IP.String()
			return
		}
	}
	err = errors.New("无法获取IP地址")
	return
}
