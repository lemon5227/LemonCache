package lru

import (
	"container/list"
)

/*
	最近最少使用，相对于仅考虑时间因素的 FIFO 和仅考虑访问频率的 LFU，
	LRU 算法可以认为是相对平衡的一种淘汰算法。LRU 认为，如果数据最近被访问过，
	那么将来被访问的概率也会更高。LRU 算法的实现非常简单，
	维护一个队列，如果某条记录被访问了，则移动到头部，
	那么队尾则是最近最少访问的数据，淘汰该条记录即可。
*/
type Cache struct {
	maxBytes int64
	nbytes int64
	ll *list.List
	cache map[string]*list.Element
	//OnEvicted 是某条记录被移除时的回调函数，可以为 nil。
	OnEvicted func(key string, value Value)
}

type entry struct {
	key string
	value Value
}

//用于返回所占用的内存大小
type Value interface{
	Len() int
}

func New(maxBytes int64,onEvited func(string,Value)) *Cache{
	return &Cache{
		maxBytes:maxBytes,
		ll:list.New(),
		cache:make(map[string]*list.Element),
		OnEvicted:onEvited,
	}
}

//查找功能,从字典中找到就给移到链表头部
func(c *Cache) Get(key string) (value Value,ok bool){
	if ele,ok := c.cache[key];ok{
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value,true
	}
	return
}

//删除功能,其实是缓存淘汰,移除最近最少访问的节点(队尾)
func(c *Cache) RemoveOldest(){
	ele := c.ll.Back()
	if ele != nil{
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache,kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil{
			c.OnEvicted(kv.key,kv.value)
		}
	}

}

//新增或修改功能,如果键存在就移到队首,如果不存在就在队首插入节点&entry{key, value},
// 并字典中添加 key 和节点的映射关系。更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点(队尾)。
func (c *Cache) Add(key string,value Value) {
	if ele,ok := c.cache[key];ok{
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key,value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.nbytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

//返回缓存中的节点数
func (c *Cache) Len() int {
	return c.ll.Len()
}