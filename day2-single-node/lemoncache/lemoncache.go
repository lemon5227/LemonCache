package LemonCache

import(
	"sync"
	"fmt"
	"log"
)

/*
一个 Group 可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name。
比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses。
第二个属性是 getter Getter，即缓存未命中时获取源数据的回调(callback)。
第三个属性是 mainCache cache，即一开始实现的并发缓存。
*/
type Group struct{
	name string
	getter Getter
	mainCache cache
}


//定义接口 Getter 和 回调函数 Get(key string)([]byte, error)，
//参数是 key，返回值是 []byte。
type Getter interface {
	Get(key string) ([]byte, error)
}

//定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法。
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte,error) {
	return f(key)
}


var (
	mu sync.RWMutex
	groups = make(map[string]*Group)	
)

//构建函数 NewGroup 用来实例化 Group，并且将 group 存储在全局变量 groups 中。
func NewGroup(name string,cacheBytes int64,getter Getter) *Group{
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		getter: getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

//GetGroup 用来特定名称的 Group，这里使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作。
func GetGroup(name string) *Group{
	mu.RLock()
	defer mu.Unlock()
	g := groups[name]
	return g
}


/*
Get 方法实现了流程 ⑴ 和 ⑶。
流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
流程 ⑶ ：缓存不存在，则调用 load 方法，load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），
getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
*/
func (g *Group) Get(key string) (ByteView,error) {
	if key =="" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	
	if v,ok := g.mainCache.get(key); ok{
		log.Println("[LemonCache] hit")
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}


func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}