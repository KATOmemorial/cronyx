package common

import (
	"net"
	"strings"
)

// GetOutboundIP 获取本机对外 IP
// 原理：并不真的连接 Google，只是利用 UDP 协议探测一下系统首选的出站 IP
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return strings.Split(localAddr.String(), ":")[0], nil
}
