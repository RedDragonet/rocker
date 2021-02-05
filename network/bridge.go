package network

import (
	"fmt"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
	"time"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		log.Errorf("创建bridge %s 失败 %v", bridgeName, err)
		return fmt.Errorf("创建bridge %s 失败 %v", bridgeName, err)
	}

	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP

	//设置 Bridge IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		log.Errorf("分配 IP 地址: %s 到 bridge: %s 失败: %v ", gatewayIP, bridgeName, err)
		return fmt.Errorf("分配 IP 地址: %s 到 bridge: %s 失败: %v ", gatewayIP, bridgeName, err)
	}
	log.Infof("分配 IP 地址: %s 到 bridge: %s ", gatewayIP, bridgeName)

	//启动 Bridge
	if err := setInterfaceUP(bridgeName); err != nil {
		log.Errorf("Bridge  %s, 启动失败: %v", bridgeName, err)
		return fmt.Errorf("Bridge  %s, 启动失败: %v", bridgeName, err)
	}
	log.Infof("Bridge  %s, 启动", bridgeName)

	//设置 IPTable
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		log.Errorf("Bridge %s IPtable 设置失败: %v ", bridgeName, err)
		return fmt.Errorf("Bridge %s IPtable 设置失败: %v ", bridgeName, err)
	}

	return nil
}

func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}
	err := d.initBridge(n)
	if err != nil {
		log.Errorf("创建 bridge 失败: %v", err)
	}

	return n, err
}

func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	//判断是否已经创建过
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("Bridge %s 不存在，无法删除 %v ", bridgeName, err)
	}

	err = netlink.LinkDel(br)
	if err != nil {
		return fmt.Errorf("Bridge %s 删除失败 %v ", bridgeName, err)
	}

	return nil
}

//链接网络和端点
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	// Veth
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + la.Name,
	}

	//创建端点 Veth
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("链接 Bridge 和端点失败: %v ", err)
	}

	//启动 Veth
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("启动  Bridge 和 端点链接: %v ", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

//创建 Bridge
func createBridgeInterface(bridgeName string) error {
	//判断是否已经创建过
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return fmt.Errorf("Bridge %s 已经存在，不能重复创建 ", bridgeName)
	}

	// create *netlink.Bridge object
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = bridgeName

	br := &netlink.Bridge{LinkAttrs: linkAttrs}
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("Bridge 创建失败 %s: %v ", bridgeName, err)
	}
	return nil
}

//启动 Interface
func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("未找到对于的 Interface [ %s ]: %v", iface.Attrs().Name, err)
	}

	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Interface 启动失败 %s: %v ", interfaceName, err)
	}
	return nil
}

// 设置 Interface 的IP
func setInterfaceIP(name string, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	// 循环多次查询，新创建的 Bridge 耗时
	for i := 0; i < retries; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Infof("获取新的 Bridge 设备失败， [ %s ]... 重试", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Errorf("setInterfaceIP: %s 查询设备失败 %v ", name, err)
		return err
	}

	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		log.Errorf("setInterfaceIP: 解析IP地址 %s 失败 %v ", rawIP, err)
		return err
	}
	addr := &netlink.Addr{IPNet: ipNet, Peer: ipNet, Label: "", Flags: 0, Scope: 0, Broadcast: nil}
	log.Infof("setInterfaceIP: %s ", addr)
	return netlink.AddrAdd(iface, addr)
}

//设置 IPTable
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables 设置失败, %v ", output)
	}
	log.Infof("driver %s , iptables  设置 %s ", bridgeName, iptablesCmd)

	iptablesCmd = fmt.Sprintf("-t nat -A POSTROUTING -o %s -m addrtype --src-type LOCAL --dst-type UNICAST -j MASQUERADE", bridgeName)
	cmd = exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err = cmd.Output()
	if err != nil {
		log.Errorf("iptables LOCALHOST 设置失败, %v ", output)
	}
	log.Infof("driver %s , iptables LOCALHOST 设置 %s ", bridgeName, iptablesCmd)
	return err
}
