package utils

import (
	"net"
	"strings"
)

// GetOutboundIP получает исходящий IP-адрес хоста
func GetOutboundIP() string {
	// Подключаемся к публичному DNS серверу для получения локального IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// ParseCIDR парсит строку CIDR и возвращает IPNet
func ParseCIDR(cidr string) (*net.IPNet, error) {
	if cidr == "" {
		return nil, nil
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	return ipNet, nil
}

// IPInCIDR проверяет, входит ли IP-адрес в CIDR подсеть
func IPInCIDR(ipStr string, cidr string) (bool, error) {
	if cidr == "" {
		return true, nil
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, nil
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}

	return ipNet.Contains(ip), nil
}

// GetRealIP извлекает реальный IP из заголовков запроса
func GetRealIP(headers map[string][]string) string {
	if xRealIP := headers["X-Real-Ip"]; len(xRealIP) > 0 && xRealIP[0] != "" {
		return xRealIP[0]
	}

	if xForwardedFor := headers["X-Forwarded-For"]; len(xForwardedFor) > 0 {
		ips := strings.Split(xForwardedFor[0], ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	return ""
}
