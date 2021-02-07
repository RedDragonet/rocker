# rocker
Docker from scratch

从零开始写一个 Docker

## 需求 
- Linux kernel > 3.12
- 暂不支持 Windows / MacOS 系统运行


## 编译
```bash
#MAC 需要安装跨平台编译库musl-cross，用于编译 CGO，适用于Linux
brew install FiloSottile/musl-cross/musl-cross
```

- make build 本地编译
- make build2remote 本地编译并发送到远程主机，远程主机IP配置在 makefile 中
    
## 配置

```bash
# 如需使用端口转发功能 (rocker run  -p 80:80)，需开启如下配置
sysctl -w net.ipv4.conf.all.forwarding=1
sysctl -w net.ipv4.conf.all.route_localnet=1
```


## 新增功能
#### 1. PULL
从 Docker-registry PULL 镜像
默认采用 [中科大镜像](https://ustc-edu-cn.mirror.aliyuncs.com)
暂不支持需要 auth 的仓库

```bash
rocker pull nginx 
# 或指定指定 registry
rocker pull http://hub-mirror.c.163.com/nginx
```



#### 2. UTS
支持 [OverlayFS](https://www.kernel.org/doc/html/latest/filesystems/overlayfs.html) 

```bash
|-- a0eec5920895888f5 容器层ID
|   |-- diff  分层文件解压
|   |   |-- layer_info
|   |   |-- root
|   |   `-- test4
|   |-- layer.tar 分层差异文件打包
|   |-- lower 下层文件目录
|   |-- merged mount overlayfs 可写入层
|   `-- work
|-- baa06c3d0434c60171d951f6edcfa7264992939a6d47bee3fbb3ee80695939cd
|   |-- diff
|   |   `-- root
|   |-- lower
|   |-- merged
```

#### 3. NSenter
支持 CGroup


```bash
rocker exec 容器id 命令
         +
         |
         v

     CLONE()                      子进程(CGO)
         +                             +
         |  +---------PIPE-----------> |
         |                             |
         |                  加入【容器id】对应的命名空间
         |                  setns() ipc/uts/net/pid/mnt
         |                             |
         |                             |
         | <----------PIPE-------------+
         |                             |
    将Clone的子进程ID                    |
    加入容器的CGroup中                   |
         | +----------PIPE------------>|
         |                             |
         |                             |
         |                         execv(【命令】)
      wait()                           v
         |
         v
```
需要用到 Cgo，原因在于 setns() 不支持多线程程序（主要是 mnt namespace 不支持），而 Go 语言的运行时为多线程。

```bash
A multithreaded process may not change user namespace with setns()
---https://man7.org/linux/man-pages/man2/setns.2.html
```

## 感谢

代码参考 https://github.com/xianlubird/mydocker