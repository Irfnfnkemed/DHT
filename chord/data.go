package chord

var mode = string("overwrite")

func Setmode(NewMode string) {
	mode = NewMode
}

func dataPut(valueOld, valueNew ValuePair) ValuePair {
	if mode == "overwrite" {
		return valueNew
	} else if mode == "append" {
		return ValuePair{(valueOld.Value + valueNew.Value), valueNew.KeyId}
	}
	return ValuePair{}
}
