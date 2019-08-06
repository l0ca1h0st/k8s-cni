**cni**

## 0x00 说明
> 这是一个没有任意意义的插件，仅仅用来学习下cni插件具体工作方式

## 0x01 前提
> 需要有一个k8s测试环境
> > 安装k8s 可以使用kubeadm 方式安装
> > `kubeadm init --kubernetes-version=v1.12.2 --pod-network-cidr=10.244.0.0/16 --apiserver-advertise-address=0.0.0.0`

## 0x02 部署
> 由于这是一个模型cni，没有具体实现ipam，使用了静态的方式， 因此需要我们做一些额外的工作
 
> * 查看k8s为每个节点分配的子网
```bash
[root@k8s-master cni]# kubectl get nodes 
NAME         STATUS   ROLES    AGE    VERSION
k8s-master   Ready    master   123m   v1.12.2
k8s-node01   Ready    <none>   122m   v1.12.2
k8s-node02   Ready    <none>   122m   v1.12.2
[root@k8s-master cni]# kubectl describe node k8s-master|grep PodCIDR
PodCIDR:                     10.244.0.0/24
[root@k8s-master cni]# kubectl describe node k8s-node01|grep PodCIDR
PodCIDR:                     10.244.1.0/24
[root@k8s-master cni]# kubectl describe node k8s-node02|grep PodCIDR
PodCIDR:                     10.244.2.0/24
[root@k8s-master cni]# 
```

> * 在每个节点上创建一个网桥
```bash
# master节点
[root@k8s-master cni]# brctl addbr cni0
[root@k8s-master cni]# ifconfig cni0 10.244.0.1/24 up

# node01 节点上
[root@k8s-node01 cni]# brctl addbr cni0
[root@k8s-node01 cni]# ifconfig cni0 10.244.1.1/24 up

# node02 节点上
[root@k8s-master cni]# brctl addbr cni0
[root@k8s-master cni]# ifconfig cni0 10.244.2.1/24 up
```

> * 在每个节点上创建cni配置文件目录
> *kubelet 会到/etc/cni/net.d/找cni网络相关配置文件*

```bash
# 在三个节点执行同样的工作
[root@k8s-master cni]# mkdir /etc/cni/net.d/

# 在/etc/cni/net.d 创建配置文件
[root@k8s-master cni]# cat /etc/cni/net.d/10-ys-plugin.conf 
{
	"cniVersion": "0.3.1",
	"name": "mynet",
	"type": "cni",
	"network": "10.244.0.0/16",
	"subnet": "10.244.0.2/24"
    # 节点1 上吧10.244.0.2/24 替换成10.244.1.2
    # 节点2 上把10.244.0.2/24 替换成10.244.2.2
}

```

> * 下载代码编译cni，并且把编译好的cni 放到/opt/cni/bin目录下


> * kubectl run 
```bash
kubectl run -i -t busybox --image=busybox --restart=Never
```
