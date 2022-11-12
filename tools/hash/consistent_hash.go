package hash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//Hash 定义计算hash的函数
type Hash func(data []byte) uint32

type Map struct {

	//自定义的hash函数
	hash Hash

	//虚拟节点数
	replicas int

	//hash环，每个节点映射的hash值
	keys []int

	//映射虚拟节点
	hashMap map[int]string
}

// New 创建一个Map实例，作为一致性hash算法的主要结构
func New(replicas int, fn Hash) *Map {
	//创建实例
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		//采用默认的hash算法
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

//Add 添加节点，可以传入一个或者多个真实节点的名称
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		//根据指定的虚拟节点数量进行创建
		for i := 0; i < m.replicas; i++ {
			//计算hash值，根据 id+key 的格式来进行不同虚拟节点的区分 （strconv.Itoa(i)格式化成string类型）
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			//将对应的hash值，添加进行
			m.keys = append(m.keys, hash)
			//映射虚拟节点和真实节点的关系
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

//Get 获取到节点信息
func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}
	//根据key计算节点的hash
	hash := int(m.hash([]byte(key)))

	//Search方法，根据keys的数量进行遍历
	idx := sort.Search(len(m.keys), func(i int) bool {
		//顺时针寻找第一个匹配的虚拟节点对应的下标，为什么要大于等于呢，因为上面对keys进行了排序，并且顺时针获取到环上面的第一个节点
		return m.keys[i] >= hash
	})
	// 如果 idx == len(m.keys) 说明应该选择 m.keys[0]，因为keys是一个环状结构，所以用取余数的方式来处理;
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
