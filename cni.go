package main

import (
	"encoding/json"
	"fmt"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"net"
	"runtime"
	"syscall"
)

type NetConf struct {
	types.NetConf

	Network string `json:"network"`
	Subnet string `json:"subnet"`
}

const (
	LINUX_BRIDEGE_NAME = "cni0"
)
func init() {
	runtime.LockOSThread()
}


func loadConfig(bytes []byte)(*NetConf, string , error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("cannot parse json file: %v", err)
	}
	return n, n.CNIVersion, nil
}



func addFunc(args *skel.CmdArgs) error {
	// 加载配置文件
	n, cniVersion, err := loadConfig(args.StdinData)
	if err != nil {
		return err
	}

	//// 创建一个linux bridget
	//// 手动创建一个cni0 ，并且赋予IP地址
	//cni0br := &netlink.Bridge{
	//	LinkAttrs: netlink.LinkAttrs{
	//		MTU:1500,
	//		Name: LINUX_BRIDEGE_NAME,
	//	},
	//}
	//// 添加cni0 接口
	//if err := netlink.LinkAdd(cni0br); err != nil {
	//	return fmt.Errorf(err.Error())
	//}
	//// 启动cni0 接口
	//if err := netlink.LinkSetUp(cni0br); err != nil {
	//	return fmt.Errorf(err.Error())
	//}
	// 查找cni0 是否已经就绪
	cni0link,err := netlink.LinkByName(LINUX_BRIDEGE_NAME)
	if err != nil && err != syscall.EEXIST {
		return fmt.Errorf(err.Error())
	}
	cni0brlink, ok := cni0link.(*netlink.Bridge)
	if ! ok{
		return fmt.Errorf("CNI0 is not a linux bridge")
	}

	// 获取CNI的namespace
	containerIface := &current.Interface{}
	hostIface := &current.Interface{}

	containerNetns, err := ns.GetNS(args.Netns)
	if err != nil {
		return err
	}

	// 在容器的namespace里面创建veth

	if err := containerNetns.Do(func(netNS ns.NetNS) error {
		hostVeth, containerVeth,err := ip.SetupVeth(args.IfName, 1500, netNS)
		if err != nil {
			return nil
		}
		containerIface.Name = containerVeth.Name
		containerIface.Mac = containerVeth.HardwareAddr.String()
		containerIface.Sandbox = containerNetns.Path()
		hostIface.Name = hostVeth.Name
		return nil
	}); err != nil {
		return err
	}


	result := &current.Result{
		CNIVersion:cniVersion,
		Interfaces: []*current.Interface{hostIface, containerIface},
	}


	// 在容器的namespace里面给veth添加网卡
	if err := containerNetns.Do(func(netNS ns.NetNS) error {

		// 获取虚拟接口
		conVethLink,err := netlink.LinkByName(containerIface.Name)
		if err != nil {
			return nil
		}
		ipv4Addr, ipv4Net,err := net.ParseCIDR(n.Subnet)
		if err != nil {
			return err
		}
		ipv4Net.IP = ipv4Addr
		containerAddr := &netlink.Addr{
			IPNet:ipv4Net,
		}

		containerIps := &current.IPConfig{
			Address: *ipv4Net,
		}

		result.IPs = []*current.IPConfig{containerIps}

		if err := netlink.AddrAdd(conVethLink, containerAddr); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// 绑定host-end 的网卡到cni0
	hostVethLink,err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return fmt.Errorf("not found host veth name for %v", err.Error())
	}
	if err := netlink.LinkSetMaster(hostVethLink, cni0brlink); err != nil {
		return fmt.Errorf("not found cni0 for %v", err.Error())
	}


	return types.PrintResult(result, cniVersion)
}

func delFunc(args *skel.CmdArgs) error {
	return nil
}

func checkFunc(args *skel.CmdArgs) error {
	return nil
}

func main() {
	skel.PluginMain(addFunc, checkFunc, delFunc, version.All, "cni plugin")
}
