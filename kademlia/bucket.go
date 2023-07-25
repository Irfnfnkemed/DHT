package kademlia

import (
	"errors"
	"sync"
)

// bucket内部的链表节点
type unit struct {
	prev *unit
	next *unit
	ip   string
}

// 每个bucket都是一个链表
type Bucket struct {
	node *Node
	head *unit
	tail *unit
	lock sync.RWMutex
	size int
	ip   string
}

// 初始化
func (bucket *Bucket) init(ip string, node *Node) error {
	bucket.lock.Lock()
	bucket.node = node
	bucket.head = new(unit)
	bucket.tail = new(unit)
	bucket.head.next, bucket.head.prev = bucket.tail, nil
	bucket.tail.next, bucket.tail.prev = nil, bucket.head
	bucket.size = 0
	bucket.ip = ip
	bucket.lock.Unlock()
	return nil
}

// 插入节点至bucket头部
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

// 将目标节点提到bucket头部
func (bucket *Bucket) shiftToHead(p *unit) {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()
	if p == nil || p == bucket.head || p == bucket.tail || p.prev == nil || p.next == nil {
		return
	}
	p.prev.next = p.next
	p.next.prev = p.prev
	p.prev = bucket.head
	p.next = bucket.head.next
	bucket.head.next.prev = p
	bucket.head.next = p
}

// 得到bucket存储ip数量
func (bucket *Bucket) getSize() int {
	bucket.lock.RLock()
	defer bucket.lock.RUnlock()
	return bucket.size
}

// 查找bucket中目标ip的位置
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

// 删除bucket中的目标ip
func (bucket *Bucket) delete(p *unit) {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()
	if p == nil || p.prev == nil || p.next == nil || p == bucket.head || p == bucket.tail {
		return
	}
	p.prev.next = p.next
	p.next.prev = p.prev
	bucket.size--
	p.prev = nil
	p.next = nil
}

// 刷新bucket中的节点分布与顺序(用于remote call之后)
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
					if !bucket.node.Ping(ipTo) {
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
}

// 给出bucket中所有的ip
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

// 检查bucket中的尾部ip是否仍然在线
func (bucket *Bucket) check() {
	bucket.lock.RLock()
	p := bucket.tail.prev
	ipTo := p.ip
	bucket.lock.RUnlock()
	if ipTo == "" {
		return
	}
	if !bucket.node.Ping(ipTo) {
		bucket.delete(p)
	} else {
		bucket.shiftToHead(p)
	}
}
