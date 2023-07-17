package kademlia

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
)

const k = 16

type unit struct {
	prev *unit
	next *unit
	ip   string
}

type Bucket struct {
	head *unit
	tail *unit
	lock sync.RWMutex
	size int
	ip   string
}

// 初始化
func (bucket *Bucket) init(ip string) error {
	bucket.lock.Lock()
	bucket.head = new(unit)
	bucket.tail = new(unit)
	bucket.head.next, bucket.head.prev = bucket.tail, nil
	bucket.tail.next, bucket.tail.prev = nil, bucket.head
	bucket.size = 0
	bucket.ip = ip
	bucket.lock.Unlock()
	return nil
}

// 插入节点至链表头部
func (bucket *Bucket) insertToHead(ip string) error {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()
	if bucket.size >= k {
		return errors.New("Max size.")
	}
	bucket.size++
	p := unit{bucket.head, bucket.head.next, ip}
	bucket.head.next.prev = &p
	bucket.head.next = &p
	return nil
}

func (bucket *Bucket) getSize() int {
	bucket.lock.RLock()
	defer bucket.lock.RUnlock()
	return bucket.size
}

// 将目标节点提到链表头部
func (bucket *Bucket) shiftToHead(p *unit) {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()
	if p == bucket.head || p == bucket.tail {
		return
	}
	p.prev.next = p.next
	p.next.prev = p.prev
	p.prev = bucket.head
	p.next = bucket.head.next
	bucket.head.next.prev = p
	bucket.head.next = p
}

func (bucket *Bucket) find(ip string) *unit {
	bucket.lock.RLock()
	defer bucket.lock.RUnlock()
	p := bucket.head.next
	for p != bucket.tail {
		if p.ip == ip {
			return p
		}
		p = p.next
	}
	return nil
}

func (bucket *Bucket) delete(p *unit) {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()
	p.prev.next = p.next
	p.next.prev = p.prev
	bucket.size--
	p.prev = nil
	p.next = nil
}

func (bucket *Bucket) flush(ip string, online bool) {
	if ip == bucket.ip {
		return
	}
	p := bucket.find(ip)
	if online {
		if p != nil {
			bucket.shiftToHead(p)
		} else {
			bucket.lock.RLock()
			size := bucket.size
			bucket.lock.RUnlock()
			if size < k {
				bucket.insertToHead(ip)
			} else {
				for i := 1; i <= size; i++ {
					bucket.lock.RLock()
					p = bucket.tail.prev
					ipTo := p.ip
					bucket.lock.RUnlock()
					if !Ping(bucket.ip, ipTo) {
						bucket.lock.Lock()
						p.ip = ip //替换节点
						bucket.lock.Unlock()
						bucket.shiftToHead(p) //移到队头
						break
					} else {
						bucket.shiftToHead(p)
					}
				}

			}
		}
	} else {
		if p != nil {
			bucket.delete(p)
		}
	}
	logrus.Infof("Node (IP = %s) flushed the k-bucket.", bucket.ip)
}

func (bucket *Bucket) getAll() []string {
	nodeList := []string{}
	bucket.lock.RLock()
	p := bucket.head.next
	for p != bucket.tail {
		nodeList = append(nodeList, p.ip)
		p = p.next
	}
	bucket.lock.RUnlock()
	return nodeList
}
