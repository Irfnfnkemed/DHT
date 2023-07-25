package kademlia

import (
	"crypto/sha1"
	"fmt"
	"math/big"
	"sync"
)

var exp [161]*big.Int

// 初始化
func initCal() {
	for i := range exp {
		exp[i] = new(big.Int).Lsh(big.NewInt(1), uint(i)) //exp[i]存储2^i
	}
}

// order链表的内部节点
type orderUnit struct {
	prev *orderUnit
	next *orderUnit
	ip   string
	dis  *big.Int
	done bool
}

// 从头部至尾部，按找离目标距离从小到大排序(用于NodeLookup和Get)
type Order struct {
	head     *orderUnit
	tail     *orderUnit
	lock     sync.RWMutex
	idTarget *big.Int
	size     int
}

// 初始化
func (order *Order) init(id *big.Int) {
	order.lock.Lock()
	order.head = new(orderUnit)
	order.tail = new(orderUnit)
	order.head.next, order.head.prev = order.tail, nil
	order.tail.next, order.tail.prev = nil, order.head
	order.size = 0
	order.idTarget = new(big.Int).Set(id)
	order.lock.Unlock()
}

// 查找order中ip所在节点
func (order *Order) find(ip string) *orderUnit {
	order.lock.RLock()
	defer order.lock.RUnlock()
	p := order.head.next
	for p != order.tail {
		if p.ip == ip {
			return p
		}
		p = p.next
	}
	return nil
}

// 按序插入order
func (order *Order) insert(ip string) {
	dis := xor(getHash(ip), order.idTarget)
	order.lock.Lock()
	defer order.lock.Unlock()
	p := order.head.next
	for p != order.tail {
		if dis.Cmp(p.dis) < 0 {
			break
		}
		p = p.next
	}
	order.size++
	newUnit := orderUnit{p.prev, p, ip, dis, false}
	p.prev.next = &newUnit
	p.prev = &newUnit
}

// 得到order中的前a个未执行查找的节点
func (order *Order) getUndoneAlpha() []*orderUnit {
	getList := []*orderUnit{}
	size := 0
	order.lock.RLock()
	p := order.head.next
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
	order.lock.RUnlock()
	return getList
}

// 得到order中所有的未执行查找的节点
func (order *Order) getUndoneAll() []*orderUnit {
	callList := []*orderUnit{}
	order.lock.RLock()
	p := order.head.next
	for p != order.tail {
		if !p.done {
			callList = append(callList, p)
		}
		p = p.next
	}
	order.lock.RUnlock()
	return callList
}

// 得到order中前k个ip
func (order *Order) getClosest() []string {
	getList := []string{}
	size := 0
	order.lock.RLock()
	p := order.head.next
	for p != order.tail {
		getList = append(getList, p.ip)
		size++
		if size == k {
			break
		}
		p = p.next
	}
	order.lock.RUnlock()
	return getList
}

// 从order中删去节点
func (order *Order) delete(p *orderUnit) {
	order.lock.Lock()
	p.next.prev = p.prev
	p.prev.next = p.next
	order.size--
	order.lock.Unlock()
}

// 根据找到的节点列表，刷新order
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

func (order *Order) A() {
	order.lock.RLock()
	p := order.head.next
	for p != order.tail {
		fmt.Println(p.ip, p.done)
		p = p.next
	}
	order.lock.RUnlock()
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
