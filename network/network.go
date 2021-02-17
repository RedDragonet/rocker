package network

import (
	"encoding/json"
	"fmt"
	"github.com/RedDragonet/rocker/container"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/rocker/network/network/"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

type Network struct {
	Name    string
	IpRange *net.IPNet
	Driver  string
}

func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}

	nwPath := path.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("error：", err)
		return err
	}
	defer nwFile.Close()

	nwJson, err := json.Marshal(nw)
	if err != nil {
		log.Errorf("error：", err)
		return err
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		log.Errorf("error：", err)
		return err
	}
	return nil
}

func (nw *Network) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(path.Join(dumpPath, nw.Name))
	}
}

func (nw *Network) load(dumpPath string) error {
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		log.Errorf("Error load nw info", err)
		return err
	}
	return nil
}

//注册已经支持 Driver ，目前只有 Bridge
//遍历所有已经配置过的网络
func Init() error {
	forwarding, err := ioutil.ReadFile("/proc/sys/net/ipv4/conf/all/forwarding")
	if err != nil {
		return err
	}
	routeLocalNet, err := ioutil.ReadFile("/proc/sys/net/ipv4/conf/all/route_localnet")
	if err != nil {
		return err
	}

	if forwarding[0] != '1' {
		log.Errorf("建议按照如下命令设置")
		log.Errorf("sysctl -w net.ipv4.conf.all.forwarding=1")
		return fmt.Errorf("forwarding 参数未配置正确")
	}
	if routeLocalNet[0] != '1' {
		log.Errorf("建议按照如下命令设置")
		log.Errorf("sysctl -w net.ipv4.conf.all.route_localnet=1")
		return fmt.Errorf("route_localnet 参数未配置正确")
	}

	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		if strings.HasSuffix(nwPath, "/") {
			return nil
		}
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		if err := nw.load(nwPath); err != nil {
			log.Errorf("遍历 Network 失败: %s", err)
		}

		networks[nwName] = nw
		return nil
	})

	return nil
}

func CreateNetwork(driver, subnet, name string) error {
	_, cidr, _ := net.ParseCIDR(subnet)
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip

	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}

	return nw.dump(defaultNetworkPath)
}

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver,
		)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("未找到对应的网络配置: %s", networkName)
	}

	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("释放 ip 地址 %s 失败 %v", nw.IpRange.IP.String(), err)
	}

	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("移除网络设备失败: %s", err)
	}

	return nw.remove(defaultNetworkPath)
}

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("未找到对应的网络配置: %s", networkName)
	}

	// 分配容器IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.ID, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.Config.PortMapping,
	}
	// 调用网络驱动挂载和配置网络端点
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	// 这里 veth 的另一端已经连接到 Bridge 了
	// ep变量中的 Device 字段也已经被赋值
	// 到容器的namespace配置容器网络设备IP地址
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	if err = container.RecordContainerIP(cinfo.ID, ip); err != nil {
		return err
	}

	return configPortMapping(ep, cinfo)
}

func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/ns/net", cinfo.State.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	runtime.LockOSThread()

	// 修改veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("set netns fd 失败, %v", err)
	}

	// 获取当前的网络namespace
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("获取当前的 namespace , %v", err)
	}

	// 设置当前进程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("set netns, %v", err)
	}
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

//配置端点IP地址和路由
func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	//找到 Veth 的另一端
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	/*****
		以下的操作都为 容器的 network namespace  中
	*****/

	defer enterContainerNetns(&peerLink, cinfo)()

	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	//配置 veth ip
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	// 启动 veth
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	// 启动回环网卡
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	//配置路由
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index, //绑定在 namespace 中的 veth
		Gw:        ep.Network.IpRange.IP,  // Bridge ip 地址
		Dst:       cidr,                   // default
	}

	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil

	/*****
		defer 执行的函数 从 network namespace 跳出
	*****/
}

func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("端口映射格式错误, %v", pm)
			continue
		}

		//判断端口占用
		iptablesCmd := fmt.Sprintf("-t nat -L PREROUTING -nv")
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		//err := cmd.Run()
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables 查询失败, %s %s %v", iptablesCmd, output, err)
			continue
		}

		if strings.Index(string(output), fmt.Sprintf("dpt:%s", portMapping[0])) != -1 {
			log.Errorf("端口 %s 已经被占用", portMapping[0])
			return fmt.Errorf("端口 %s 已经被占用", portMapping[0])
		}

		iptablesCmd = fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s ! -i lo -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd = exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		//err := cmd.Run()
		output, err = cmd.Output()
		if err != nil {
			log.Errorf("iptables 设置失败, %v", output)
			continue
		}
		log.Infof("端口映射 %s 设置 %s ", pm, iptablesCmd)

		//localhost 端口映射

		iptablesCmd = fmt.Sprintf("-t nat -A OUTPUT -o lo -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd = exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		//err := cmd.Run()
		output, err = cmd.Output()
		if err != nil {
			log.Errorf("iptables localhost 设置失败, %v", output)
			continue
		}
		log.Infof("端口映射 localhost %s 设置 %s ", pm, iptablesCmd)
	}
	return nil
}
