package core

import (
	"Spark/modules"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/denisbrodbeck/machineid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	_net "net"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"
)

func isPrivateIP(ip _net.IP) bool {
	var privateIPBlocks []*_net.IPNet
	for _, cidr := range []string{
		//"127.0.0.0/8",    // IPv4 loopback
		//"::1/128",        // IPv6 loopback
		//"fe80::/10",      // IPv6 link-local
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
	} {
		_, block, _ := _net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func GetLocalIP() (string, error) {
	ifaces, err := _net.Interfaces()
	if err != nil {
		return `Unknown`, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return `Unknown`, err
		}

		for _, addr := range addrs {
			var ip _net.IP
			switch v := addr.(type) {
			case *_net.IPNet:
				ip = v.IP
			case *_net.IPAddr:
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
	interfaces, err := _net.Interfaces()
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

func GetNetIOInfo() (modules.Net, error) {
	result := modules.Net{}
	first, err := net.IOCounters(false)
	if err != nil {
		return result, nil
	}
	if len(first) == 0 {
		return result, errors.New(`failed to read network io counters`)
	}
	<-time.After(time.Second)
	second, err := net.IOCounters(false)
	if err != nil {
		return result, nil
	}
	if len(second) == 0 {
		return result, errors.New(`failed to read network io counters`)
	}
	result.Recv = second[0].BytesRecv - first[0].BytesRecv
	result.Sent = second[0].BytesSent - first[0].BytesSent
	return result, nil
}

func GetCPUInfo() (modules.CPU, error) {
	result := modules.CPU{}
	info, err := cpu.Info()
	if err != nil {
		return result, nil
	}
	if len(info) == 0 {
		return result, errors.New(`failed to read cpu info`)
	}
	result.Model = info[0].ModelName
	result.Cores.Logical, _ = cpu.Counts(true)
	result.Cores.Physical, _ = cpu.Counts(false)
	stat, err := cpu.Percent(3*time.Second, false)
	if err != nil {
		return result, nil
	}
	if len(stat) == 0 {
		return result, errors.New(`failed to read cpu info`)
	}
	result.Usage = stat[0]
	return result, nil
}

func GetRAMInfo() (modules.IO, error) {
	result := modules.IO{}
	stat, err := mem.VirtualMemory()
	if err != nil {
		return result, nil
	}
	result.Total = stat.Total
	result.Used = stat.Used
	result.Usage = float64(stat.Used) / float64(stat.Total) * 100
	return result, nil
}

func GetDiskInfo() (modules.IO, error) {
	result := modules.IO{}
	disk.IOCounters()
	disks, err := disk.Partitions(true)
	if err != nil {
		return result, nil
	}
	for i := 0; i < len(disks); i++ {
		stat, err := disk.Usage(disks[i].Mountpoint)
		if err == nil {
			result.Total += stat.Total
			result.Used += stat.Used
		}
	}
	result.Usage = float64(result.Used) / float64(result.Total) * 100
	return result, nil
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
	localIP, err := GetLocalIP()
	if err != nil {
		localIP = `unknown`
	}
	macAddr, err := GetMacAddress()
	if err != nil {
		macAddr = `unknown`
	}
	cpuInfo, err := GetCPUInfo()
	if err != nil {
		cpuInfo = modules.CPU{
			Model: `unknown`,
			Usage: 0,
		}
	}
	netInfo, err := GetNetIOInfo()
	if err != nil {
		netInfo = modules.Net{
			Sent: 0,
			Recv: 0,
		}
	}
	ramInfo, err := GetRAMInfo()
	if err != nil {
		ramInfo = modules.IO{
			Total: 0,
			Used:  0,
			Usage: 0,
		}
	}
	diskInfo, err := GetDiskInfo()
	if err != nil {
		diskInfo = modules.IO{
			Total: 0,
			Used:  0,
			Usage: 0,
		}
	}
	uptime, err := host.Uptime()
	if err != nil {
		uptime = 0
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = `unknown`
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
	return &modules.Device{
		ID:       id,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		LAN:      localIP,
		MAC:      macAddr,
		CPU:      cpuInfo,
		RAM:      ramInfo,
		Net:      netInfo,
		Disk:     diskInfo,
		Uptime:   uptime,
		Hostname: hostname,
		Username: username.Username,
	}, nil
}

func GetPartialInfo() (*modules.Device, error) {
	cpuInfo, err := GetCPUInfo()
	if err != nil {
		cpuInfo = modules.CPU{
			Model: `unknown`,
			Usage: 0,
		}
	}
	netInfo, err := GetNetIOInfo()
	if err != nil {
		netInfo = modules.Net{
			Recv: 0,
			Sent: 0,
		}
	}
	memInfo, err := GetRAMInfo()
	if err != nil {
		memInfo = modules.IO{
			Total: 0,
			Used:  0,
			Usage: 0,
		}
	}
	diskInfo, err := GetDiskInfo()
	if err != nil {
		diskInfo = modules.IO{
			Total: 0,
			Used:  0,
			Usage: 0,
		}
	}
	uptime, err := host.Uptime()
	if err != nil {
		uptime = 0
	}
	return &modules.Device{
		Net:    netInfo,
		CPU:    cpuInfo,
		RAM:    memInfo,
		Disk:   diskInfo,
		Uptime: uptime,
	}, nil
}
