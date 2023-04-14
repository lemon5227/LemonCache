package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 定义了函数类型 Hash，采取依赖注入的方式，
// 允许用于替换成自定义的 Hash 函数，也方便测试时替换，默认为 crc32.ChecksumIEEE 算法
type Hash func(data []byte) uint32

type Map struct {
	hash Hash
	replicas int //虚拟节点倍数
	keys     []int // Sorted
	hashMap  map[int]string //虚拟节点与真实节点的映射表 hashMap，键是虚拟节点的哈希值，值是真实节点的名称。
}

func New(replicas int,fn Hash) 	*Map{
	m := &Map{
		hash: fn,
		replicas: replicas,
		hashMap: make(map[int]string),
	}
	if m.hash == nil{
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

//Add 函数允许传入 0 或 多个真实节点的名称。
func (m *Map) Add(keys...string) {
	for _,key := range keys {
		for i:= 0; i< m.replicas; i++ {
			//对每一个真实节点 key，对应创建 m.replicas 个虚拟节点，
			//虚拟节点的名称是：strconv.Itoa(i) + key，即通过添加编号的方式区分不同虚拟节点。
			hash := int(m.hash([]byte(strconv.Itoa(i)+key)))
			//使用 m.hash() 计算虚拟节点的哈希值，使用 append(m.keys, hash) 添加到环上。
			m.keys = append(m.keys,hash)
			//在 hashMap 中增加虚拟节点和真实节点的映射关系。
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	//sort.Search(n int,func)会在[0,n)中查找第一个满足f(i)==true的i，如果不存在这样的i，返回n。
	//这里的f(i)是m.keys[i] >= hash，即查找第一个大于等于 hash 的 m.keys[i]。
	//如果不存在这样的 m.keys[i]，则 idx 会等于 len(m.keys)，即 idx%len(m.keys) 会等于 0，
	idx := sort.Search(len(m.keys),func(i int) bool{
		return m.keys[i] >= hash
	}) 
	return m.hashMap[m.keys[idx%len(m.keys)]]
}