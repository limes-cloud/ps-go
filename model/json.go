package model

import "github.com/json-iterator/go"

// 使用此包代替原生json，性能快10x
var json = jsoniter.ConfigCompatibleWithStandardLibrary
