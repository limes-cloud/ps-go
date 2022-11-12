package hash

import (
	"fmt"
	"ps-go/consts"
)

var hashObject *Map

// Init 初始化一致性hash
func Init() {
	hashObject = New(consts.MaxLogReplicaCount, nil)
	for i := 0; i < consts.MaxLogReplicaCount; i++ {
		hashObject.Add(fmt.Sprint(i))
	}
}

func GetHash() *Map {
	return hashObject
}
