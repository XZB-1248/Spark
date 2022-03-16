package core

import (
	"Spark/modules"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/denisbrodbeck/machineid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"net"
	"os"
	"os/user"
	"runtime"
	"strings"
)

func isPrivateIP(ip net.IP) bool {
	var privateIPBlocks []*net.IPNet
	for _, cidr := range []string{
		//"127.0.0.0/8",    // IPv4 loopback
		//"::1/128",        // IPv6 loopback
		//"fe80::/10",      // IPv6 link-local
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func GetCPUInfo() (string, error) {
	info, err := cpu.Info()
	if err != nil {
		return ``, nil
	}
	if len(info) > 0 {
		return info[0].ModelName, nil
	}
	return ``, errors.New(`failed to read cpu info`)
}

func GetLocalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return `Unknown`, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return `Unknown`, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if isPrivateIP(ip) {
				if addr := ip.To4(); addr != nil {
					return addr.String(), nil
				} else if addr := ip.To16(); addr != nil {
					return addr.String(), nil
				}
			}
		}
	}
	return `Unknown`, errors.New(`no IP address found`)
}

func GetMacAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ``, err
	}
	var address []string
	for _, i := range interfaces {
		a := i.HardwareAddr.String()
		if a != `` {
			address = append(address, a)
		}
	}
	if len(address) == 0 {
		return ``, nil
	}
	return strings.ToUpper(address[0]), nil
}

func GetMemSize() (uint64, error) {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, nil
	}
	return memStat.Total, nil
}

func GetDevice() (*modules.Device, error) {
	id, err := machineid.ProtectedID(`Spark`)
	if err != nil {
		id, err = machineid.ID()
		if err != nil {
			secBuffer := make([]byte, 16)
			rand.Reader.Read(secBuffer)
			id = hex.EncodeToString(secBuffer)
		}
	}
	cpuModel, err := GetCPUInfo()
	if err != nil {
		cpuModel = `unknown`
	}
	localIP, err := GetLocalIP()
	if err != nil {
		localIP = `unknown`
	}
	macAddr, err := GetMacAddress()
	if err != nil {
		macAddr = `unknown`
	}
	memSize, err := GetMemSize()
	if err != nil {
		memSize = 0
	}
	uptime, err := host.Uptime()
	if err != nil {
		uptime = 0
	}
	username, err := user.Current()
	if err != nil {
		username = &user.User{Username: `unknown`}
	} else {
		slashIndex := strings.Index(username.Username, `\`)
		if slashIndex > -1 && slashIndex+1 < len(username.Username) {
			username.Username = username.Username[slashIndex+1:]
		}
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = `unknown`
	}
	return &modules.Device{
		ID:       id,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPU:      cpuModel,
		LAN:      localIP,
		Mac:      macAddr,
		Mem:      memSize,
		Uptime:   uptime,
		Hostname: hostname,
		Username: username.Username,
	}, nil
}
