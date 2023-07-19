# Distributed Hash Table - PPCA 2023

## 文件结构
```
│─── naive
│    └── node.go  
├── chord
│    ├── chord.go
│    ├── rpcWrapper.go
│    └── tool.go
├── kademlia
│    ├── bucket.go
│    ├── data.go
│    ├── kademlia.go
│    ├── rpcWrapper.go
│    └── tool.go
├── rpc
│    └── rpc.go
├── test
│    ├── advance.go
│    ├── basic.go
│    ├── interface.go
│    ├── test.go
│    ├── userdef.go
│    └── utils.go
├── doc
│    ├── DHT.md
│    ├── env-setup.mu
│    ├── interface.go
│    ├── report.md
│    └── tutorial.md
├── .gitignore
├── README.md
├── go.mod
├── go.sum
└── main.go
```

## 相关内容 

[项目综述与要求](doc/DHT.md)

* [环境配置](doc/env-setup.md)
* [学习材料](doc/tutorial.md)

[报告](doc/report.md)



## 测试方式：
输入
```bash
go build
./dht -protocol PROTOCOL_NAME -test TEST_PART 
```
其中，`PROTOCOL_NAME`为`naive`/`chord`/`kademlia`，`TEST_PART`为`basic`/`advance`/`all`。