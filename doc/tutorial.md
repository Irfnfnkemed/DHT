# Tutorial

本文档给出一些学习建议和推荐资料。

## 关于 Go 语法

建议先快速浏览 Go 语言官方教程 [A Tour of Go](https://go.dev/tour/list)。留个印象即可，之后可以边写边查相关语法。

你需要**重点**掌握的是 [goroutine 并发编程](https://chai2010.cn/advanced-go-programming-book/ch1-basic/ch1-06-goroutine.html) 和 [RPC 远程过程调用](https://chai2010.cn/advanced-go-programming-book/ch4-rpc/ch4-01-rpc-intro.html)。


## 关于测试程序

助教下发（并用以评分）的测试程序所做的事情是：在你的电脑上初始化若干个 DHT 节点，通过网络传输信息并维护结构，用以模拟在若干台分布的服务器运行效果。
你**不应**通过网络以外的途径通信（例如内存），你的程序**应当**能在真正的分布式场景下正确运行。

## 关于 DHT 协议

推荐阅读的综述 [blog](https://luyuhuang.tech/2020/03/06/dht-and-p2p.html)，对协议有个初步了解。

详细技术细节请翻阅两篇 paper：[Chord](https://pdos.csail.mit.edu/papers/chord:sigcomm01/chord_sigcomm.pdf) 和 [Kademlia](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf) (助教非常**希望**每位同学都阅读论文)

其他辅助参考资料：[Kademlia in Go](http://blog.notdot.net/tag/kademlia)，[Chord 解释](https://zhuanlan.zhihu.com/p/53711866)，[Kademlia 解释](http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#intro)。

## 关于 Debug

建议不要使用单步调试去调试整个 DHT（这是因为 DHT 依赖于时延，而单步调试会改变时延，实际运行结果与单步调试的结果不相同）。

推荐 Debug 方式：使用 [logrus](https://github.com/sirupsen/logrus) 库将每个节点的行为记录下来并分析。

生成的 log 文件可能非常大（几百 MB），许多文本编辑器不能正常地打开和浏览它。这里推荐使用 [Klogg](https://klogg.filimonov.dev/) 软件浏览 log。

## 关于 Application

欢迎任何天马行空的想法。以下是示例：

一个好玩的 [网站](https://iknowwhatyoudownload.com/)：嗅探 BT 网络中 IP 的资源请求。

P2P 资源分享应用：[BitTorrent](https://blog.jse.li/posts/torrent/#putting-it-all-together) 及一些原理解释 [blog](https://www.cnblogs.com/LittleHann/p/6180296.html)。
