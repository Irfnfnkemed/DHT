package kademlia

import (
	"errors"

	"github.com/sirupsen/logrus"
)

const k = 8

type unit struct {
	prev *unit
	next *unit
	ip   string
}

type Bucket struct {
	head *unit
	tail *unit
	size int
	ip   string
}

// 初始化
func (bucket *Bucket) init(ip string) error {
	bucket.head = new(unit)
	bucket.tail = new(unit)
	bucket.head.next, bucket.head.prev = bucket.tail, nil
	bucket.tail.next, bucket.tail.prev = nil, bucket.head
	bucket.size = 0
	bucket.ip = ip
	return nil
}

// 插入节点至链表头部
func (bucket *Bucket) insertToHead(ip string) error {
	if bucket.size >= k {
		return errors.New("Max size.")
	}
	p := unit{bucket.head, bucket.head.next, ip}
	bucket.head.next.prev = &p
	bucket.head.next = &p
	bucket.size++
	return nil
}

func (bucket *Bucket) getSize() int {
	return bucket.size
}

// 将目标节点提到链表头部
func (bucket *Bucket) shiftToHead(p *unit) {
	p.prev.next = p.next
	p.next.prev = p.prev
	p.prev = bucket.head
	p.next = bucket.head.next
	bucket.head.next.prev = p
	bucket.head.next = p
}

func (bucket *Bucket) find(ip string) *unit {
	p := bucket.head.next
	for p != bucket.tail {
		if p.ip == ip {
			return p
		}
		p = p.next
	}
	return nil
}

func (bucket *Bucket) begin() *unit {
	return bucket.head.next
}

func (bucket *Bucket) end() *unit {
	return bucket.tail
}

func (bucket *Bucket) flush(ip string) {
	p := bucket.find(ip)
	if p != nil {
		bucket.shiftToHead(p)
	} else if bucket.size < k {
		bucket.insertToHead(ip)
	} else {
		p = bucket.tail.prev
		for p != bucket.head {
			if !Ping(bucket.ip, p.ip) {
				p.ip = ip             //替换节点
				bucket.shiftToHead(p) //移到队头
				break
			}
			p = p.prev
		}
	}
	logrus.Infof("Node (IP = %s) flushed the k-bucket.", bucket.ip)
}
