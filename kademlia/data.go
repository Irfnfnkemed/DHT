package kademlia

import (
	"sync"
	"time"
)

type DataPair struct {
	Key   string
	Value string
}

type Data struct {
	dataPair      map[string]string
	dataLock      sync.RWMutex
	republishTime map[string]time.Time //重新发布时间
	abandonTime   map[string]time.Time //舍弃时间
}

const RepublishTime = 15 * time.Second
const AbandonTime = 40 * time.Second

// 初始化
func (data *Data) Init() {
	data.dataLock.Lock()
	data.dataPair = make(map[string]string)
	data.republishTime = make(map[string]time.Time)
	data.abandonTime = make(map[string]time.Time)
	data.dataLock.Unlock()
}

// 给出所有需要重新发布的数据
func (data *Data) getRepublishList() []DataPair {
	republishList := []DataPair{}
	data.dataLock.RLock()
	for key, republishTime := range data.republishTime {
		if time.Now().After(republishTime) {
			republishList = append(republishList, DataPair{key, data.dataPair[key]})
		}
	}
	data.dataLock.RUnlock()
	return republishList
}

// 查找数据
func (data *Data) get(key string) (string, bool) {
	data.dataLock.RLock()
	value, ok := data.dataPair[key]
	data.dataLock.RUnlock()
	return value, ok
}

// 存入数据
func (data *Data) put(key, value string) {
	data.dataLock.Lock()
	data.dataPair[key] = value
	data.republishTime[key] = time.Now().Add(RepublishTime)
	data.abandonTime[key] = time.Now().Add(AbandonTime)
	data.dataLock.Unlock()
}

// 舍弃过期数据
func (data *Data) abandon() {
	keyList := []string{}
	data.dataLock.Lock()
	for key, abandonTime := range data.abandonTime {
		if time.Now().After(abandonTime) {
			keyList = append(keyList, key)
		}
	}
	for _, key := range keyList {
		delete(data.dataPair, key)
		delete(data.republishTime, key)
		delete(data.abandonTime, key)
	}
	data.dataLock.Unlock()
}
