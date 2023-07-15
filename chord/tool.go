package chord

import (
	"crypto/sha1"
	"math/big"
)

var exp [161]*big.Int

// 得到hash值
func getHash(ip string) *big.Int {
	hash := sha1.Sum([]byte(ip))
	hashInt := new(big.Int)
	return hashInt.SetBytes(hash[:])
}

// 判断是否在目标区间内
func belong(leftOpen, rightOpen bool, beg, end, tar *big.Int) bool {
	cmpBegEnd, cmpTarBeg, cmpTarEnd := beg.Cmp(end), tar.Cmp(beg), tar.Cmp(end)
	if cmpBegEnd == -1 {
		if cmpTarBeg == -1 || cmpTarEnd == 1 {
			return false
		} else if cmpTarBeg == 1 && cmpTarEnd == -1 {
			return true
		} else if cmpTarBeg == 0 {
			return leftOpen
		} else if cmpTarEnd == 0 {
			return rightOpen
		}
	} else if cmpBegEnd == 1 {
		if cmpTarBeg == -1 && cmpTarEnd == 1 {
			return false
		} else if cmpTarBeg == 1 || cmpTarEnd == -1 {
			return true
		} else if cmpTarBeg == 0 {
			return leftOpen
		} else if cmpTarEnd == 0 {
			return rightOpen
		}
	} else if cmpBegEnd == 0 { //两端点重合
		if cmpTarBeg == 0 {
			return leftOpen || rightOpen
		} else {
			return true
		}
	}
	return false
}

func initCal() {
	for i := range exp {
		exp[i] = new(big.Int).Lsh(big.NewInt(1), uint(i)) //exp[i]存储2^i
	}
}

// 计算n+2^i并对2^160取模
func cal(n *big.Int, i int) *big.Int {
	tmp := new(big.Int).Add(n, exp[i])
	if tmp.Cmp(exp[160]) >= 0 {
		tmp.Sub(tmp, exp[160])
	}
	return tmp
}
