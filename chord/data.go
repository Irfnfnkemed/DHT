package chord

import (
	"math/big"
)

// 提供两种数据存储模式： 覆盖overwrite/添加append
const OVERWRITE = "a"
const APPEND = "b"

func GetMode(NewMode string) string {
	if NewMode == "overwrite" {
		return OVERWRITE
	} else if NewMode == "append" {
		return APPEND
	}
	return ""
}

func dataPut(valueOld, valueNew ValuePair, mode byte) ValuePair {
	if mode == 'a' {
		return valueNew
	} else if mode == 'b' {
		return ValuePair{(valueOld.Value + valueNew.Value), new(big.Int).Set(valueNew.KeyId)}
	}
	return ValuePair{}
}
