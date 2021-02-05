package network

import (
	"encoding/json"
	"fmt"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"net"
	"os"
	"path"
	"strings"
)

const ipamDefaultAllocatorPath = "/var/run/rocker/network/ipam/subnet.json"

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

func (ipam *IPAM) load() error {
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
		return err
	}
	return nil
}

func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(ipamConfigFileDir, 0644)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}

	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}

	return nil
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 存放网段中地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	// 从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("获取 IPAM 配置信息错误, %v", err)
		err = fmt.Errorf("获取 IPAM 配置信息错误, %v", err)
		return
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	// 掩码前部位数和总位数
	leading, size := subnet.Mask.Size()

	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		//可以被分配的IP总数 (2^剩余位数)
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-leading))
	}
	for c := range ((*ipam.Subnets)[subnet.String()]) {
		//循环找到为被分配的位置
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			//超出可分配上限（-2 为 0/255 网关/广播 地址不允许分配）
			if len(ipalloc)-2 < c {
				log.Errorf("超出可分配IP上限 %d", len(ipalloc))
				err = fmt.Errorf("超出可分配IP上限 %d", len(ipalloc))
				return
			}
			//空位置1
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			ip = subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				//从 HOST ID => Network ID 循环处理每一段，如    0 | 0 | 168 | 192
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			//可用地址从1开始（）
			ip[3] += 1
			break
		}
	}

	err = ipam.dump()
	if err != nil {
		log.Errorf("保存分配的IP失败, %v", err)
		err = fmt.Errorf("保存分配的IP失败, %v", err)
		return
	}
	return
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	err := ipam.load()
	if err != nil {
		log.Errorf("获取 IPAM 配置信息错误, %v", err)
		return fmt.Errorf("获取 IPAM 配置信息错误, %v", err)
	}

	c := 0
	releaseIP := ipaddr.To4()
	//可用地址从1开始，减去1
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		//找到c 在 ipalloc 的位置
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	err = ipam.dump()
	if err != nil {
		log.Errorf("释放分配的IP失败, %v", err)
		return fmt.Errorf("释放分配的IP失败, %v", err)
	}
	return nil
}
