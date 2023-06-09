package LemonCache

import(
	"sync"
	"fmt"
	"log"
	"LemonCache/singleflight"
	pb "LemonCache/lemoncachepb"
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
	peers PeerPicker
	//使用singleflight确保每个Key只被取走一次避免缓存击穿
	loader *singleflight.Group
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
		loader: &singleflight.Group{},
	}
	groups[name] = g
	return g
}

//GetGroup 用来特定名称的 Group，这里使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作。
func GetGroup(name string) *Group{
	mu.RLock()
	defer mu.RUnlock()
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
	//每个key只取一次(无论是本地还是远程)
	//无论同时并发调用多少次
	view,err := g.loader.Do(key,func()(interface{},error){
		if g.peers != nil{
			if peer,ok := g.peers.PickPeer(key);ok{
				if value,err = g.getFromPeer(peer,key);err == nil{
					return value,nil
				}
				log.Println("[LemonCache] Failed to get from peer",err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil{
		return view.(ByteView),nil
	}
	return
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

//将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中。
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter,key string) (ByteView,error) {
	req := &pb.Request{
		Group: g.name,
		Key: key,
	}
	res := &pb.Response{}
	err := peer.Get(req,res)
	if err != nil {
		return ByteView{},err
	}
	return ByteView{b:res.Value},nil
}

