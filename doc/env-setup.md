# 环境配置文档

以下配置在 Ubuntu 22.04 下测试通过。

**注意：测试程序在 WSL 1 上无法正常运行**。请根据微软文档 [Check which version of WSL you are running](https://learn.microsoft.com/en-us/windows/wsl/install#check-which-version-of-wsl-you-are-running)，
如果你正在使用 WSL 1，请 [Upgrade version from WSL 1 to WSL 2](https://learn.microsoft.com/en-us/windows/wsl/install#upgrade-version-from-wsl-1-to-wsl-2)。

## 安装 Go

本项目需要 Go 1.18 或以上版本。

删除旧版本 Go：

```bash
sudo rm -rf /usr/local/go
```

下载 Go 安装包：（你可以从 [Go 官网](https://go.dev/dl/) 获取最新版本下载链接）

```bash
curl -LO "https://go.dev/dl/go1.20.5.linux-amd64.tar.gz"
```

解压到 `/usr/local` 目录：

```bash
sudo tar -C /usr/local -xzf go1.20.5.linux-amd64.tar.gz
```

将 `/usr/local/go/bin` 目录添加到 PATH 环境变量中。

如果你使用的是 zsh，执行以下命令：

```bash
echo 'export PATH="$PATH:/usr/local/go/bin"' >> ~/.zshrc
```

如果你使用的是 bash，执行以下命令：

```bash
echo 'export PATH="$PATH:/usr/local/go/bin"' >> ~/.bashrc
```

重启终端使环境变量生效。

之后运行以下命令检查 Go 版本：

```bash
go version
```

配置 [Go 模块代理](https://goproxy.cn/) 加速 Go 模块的下载：

```bash
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
```

## 配置 VSCode 开发环境

> 推荐使用 VSCode 作为开发工具。你也可以使用 GoLand 等其他 IDE，但是请自行解决环境配置问题。

> 如果你使用虚拟机或服务器，建议使用 VSCode 的 Remote - SSH 插件连接到虚拟机或服务器上进行开发。

安装 [VSCode Go 语言扩展](https://marketplace.visualstudio.com/items?itemName=golang.go)。

安装后会弹出若干次缺包提示，内容类似于

```plain
The "gopls" command is not available. Run "go install -v golang.org/x/tools/gopls@latest" to install.
```

一律选择 Install。

## 编译测试程序

进入项目根目录，执行以下命令：

```bash
go build
```

如果编译成功，会在当前目录下生成名为 `dht` 的可执行文件，这表示你的环境配置成功。

你可以通过以下命令运行测试程序：

```bash
./dht -test all
```

测试需要一段时间，正常情况下你会看到测试通过的提示。你还会在当前目录下看到名为 `dht-test.log` 的日志文件，其中包含了测试程序的运行日志。

```plain
Final print:
Basic test passed with fail rate 0.0000
Force quit test passed with fail rate 0.0000
Quit & Stabilize test passed with fail rate 0.0000
```

如果你遇到 `Too many open files` 的错误，可以参考 [资源释放](#资源释放) 部分。

## 资源释放

**注意：这部分配置不是必须的**，但是如果你测试程序运行时遇到问题，可以尝试以下配置。

释放本项⽬需要的⼀些资源限制，包括 Port Range 和 Max File 和 TCP MSL。

```bash
sudo vim /etc/systcl.conf # 在该⽂件尾部添加下⾏
net.ipv4.ip_local_port_range = 20240 65535
net.ipv4.tcp_fin_timeout = 4
```

```bash
sudo vim /etc/security/limits.conf # 在该⽂件尾部添加下⾯⼏⾏
* soft nofile 65535
* hard nofile 65535
```

```bash
reboot # 重启
```

## 参考资料

- [mac OS下的资源限制 以及 引出的ulimit, launchctl, sysctl区别](https://blog.csdn.net/Lockheed_Hong/article/details/75258600)
- [TCP/IP中MSL详解](https://blog.51cto.com/u_10706198/1775555)，
- [GO111MODULE 是个啥？](https://zhuanlan.zhihu.com/p/374372749)
