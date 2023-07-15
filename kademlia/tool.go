package kademlia

import (
	"crypto/sha1"
	"math/big"
)

type Null struct{}

var exp [161]*big.Int

const a = 3

func initCal() {
	for i := range exp {
		exp[i] = new(big.Int).Lsh(big.NewInt(1), uint(i)) //exp[i]存储2^i
	}
}

// 得到hash值
func getHash(ip string) *big.Int {
	hash := sha1.Sum([]byte(ip))
	hashInt := new(big.Int)
	return hashInt.SetBytes(hash[:])
}

// 得到异或值
func xor(id1, id2 *big.Int) *big.Int {
	return new(big.Int).Xor(id1, id2)
}

// 给出idTo在idFrom节点中所属的bucket编号
func belong(idFrom, idTo *big.Int) int {
	dis := xor(idFrom, idTo)
	for i := 159; i >= 0; i-- {
		if dis.Cmp(exp[i]) >= 0 {
			return i
		}
	}
	return -1 //表明idFrom和idTo相同
}

type orderUnit struct {
	prev *orderUnit
	next *orderUnit
	ip   string
	dis  *big.Int
	done bool
}

// 从头部至尾部，按找离目标距离从小到大排序
type Order struct {
	head     *orderUnit
	tail     *orderUnit
	idTarget *big.Int
	size     int
}

func (order *Order) init(ipTarget string) {
	order.head = new(orderUnit)
	order.tail = new(orderUnit)
	order.head.next, order.head.prev = order.tail, nil
	order.tail.next, order.tail.prev = nil, order.head
	order.size = 0
	order.idTarget = getHash(ipTarget)
}

func (order *Order) find(ip string) *orderUnit {
	p := order.head.next
	for p != order.tail {
		if p.ip == ip {
			return p
		}
		p = p.next
	}
	return nil
}

func (order *Order) insert(ip string) {
	dis := xor(getHash(ip), order.idTarget)
	p := order.head.next
	for p != order.tail {
		if dis.Cmp(p.dis) < 0 {
			break
		}
		p = p.next
	}
	newUnit := orderUnit{p.prev, p, ip, dis, false}
	p.prev.next = &newUnit
	p.prev = &newUnit
}

func (order *Order) get() []*orderUnit {
	p := order.head.next
	getList := []*orderUnit{}
	size := 0
	for p != order.tail {
		if !p.done {
			getList = append(getList, p)
			size++
		}
		if size == a {
			break
		}
		p = p.next
	}
	return getList
}

func (order *Order) flush(findList []string) bool {
	flag := false
	for _, ipFind := range findList {
		p := order.find(ipFind)
		if p == nil {
			flag = true
			order.insert(ipFind)
		}
	}
	return flag
}

func (order *Order) getUndone() []*orderUnit {
	callList := []*orderUnit{}
	p := order.head.next
	for p != order.tail {
		if !p.done {
			callList = append(callList, p)
		}
		p = p.next
	}
	return callList
}

func (order *Order) getClosest() []string {
	getList := []string{}
	p := order.head.next
	size := 0
	for p != order.tail {
		getList = append(getList, p.ip)
		size++
		if size == k {
			break
		}
		p = p.next
	}
	return getList
}
