package kademlia

import (
	"crypto/sha1"
	"math/big"
)

type Null struct{}

var exp [161]*big.Int

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
	return 0
}
