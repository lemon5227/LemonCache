package LemonCache

import (
	"LemonCache/consistenthash"
	"fmt"
	pb "LemonCache/lemoncachepb"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"google.golang.org/protobuf/proto"
)


const defaultBasePath = "/_lemoncache/"
const defaultReplicas = 50

type HTTPPool struct {
	self string // 本机地址，包括端口
	basePath string // 节点前缀
	mu sync.Mutex // 保护 peers 和 httpGetters
	//成员变量 peers，类型是一致性哈希算法的 Map，用来根据具体的 key 选择节点。
	// 节点映射表，键是节点的 HTTP 地址，值是节点的名字
	peers *consistenthash.Map 
	// 节点与对应的 HTTP 客户端映射表
	httpGetters map[string]*HTTPGetter 
}




func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self: self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string,v ...interface{}){
	log.Printf("[Server %s] %s",p.self,fmt.Sprintf(format,v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter,r *http.Request) {
	if !strings.HasPrefix(r.URL.Path,p.basePath){
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s",r.Method,r.URL.Path)
	// 节点前缀后面的字符串以/分割，分割成两部分子字符串，最后一个字符串为未分割的余数
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view,err := group.Get(key)
	if err != nil {
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}

	body,err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)


}

func (p *HTTPPool) Set(peers...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas,nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*HTTPGetter,len(peers))
	for _,peer := range peers {
		p.httpGetters[peer] = &HTTPGetter{baseURL: peer + p.basePath}
	}
}

//实现 PeerPicker 接口的 PickPeer() 方法
func (p *HTTPPool) PickPeer(key string) (peer PeerGetter, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s",peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)


//创建具体的 HTTP 客户端类 httpGetter
type HTTPGetter struct {
	baseURL string
}

// //实现 PeerGetter 接口的 Get() 方法
func (h *HTTPGetter) Get(in *pb.Request,out *pb.Response) (error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.Group),
		url.QueryEscape(in.Key),
	)
	//发送 HTTP GET 请求
	res,err := http.Get(u)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v",res.Status)
	}
	//读取响应的数据
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v",err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil
}

var _ PeerGetter = (*HTTPGetter)(nil)

