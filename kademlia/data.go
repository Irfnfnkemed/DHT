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
	republishTime map[string]time.Time
	abandonTime   map[string]time.Time
}

func (data *Data) Init() {
	data.dataLock.Lock()
	data.dataPair = make(map[string]string)
	data.republishTime = make(map[string]time.Time)
	data.abandonTime = make(map[string]time.Time)
	data.dataLock.Unlock()
}

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

func (data *Data) get(key string) (string, bool) {
	data.dataLock.RLock()
	value, ok := data.dataPair[key]
	data.dataLock.RUnlock()
	return value, ok
}

func (data *Data) put(key, value string) {
	data.dataLock.RLock()
	data.dataPair[key] = value
	data.republishTime[key] = time.Now().Add(10 * time.Second)
	data.abandonTime[key] = time.Now().Add(15 * time.Second)
	data.dataLock.RUnlock()
}

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

func (data *Data) flush(dataPair DataPair) {
	data.dataLock.Lock()
	data.dataPair[dataPair.Key] = dataPair.Value
	data.republishTime[dataPair.Key] = time.Now().Add(10 * time.Second)
	data.abandonTime[dataPair.Key] = time.Now().Add(15 * time.Second)
	data.dataLock.Unlock()
}
