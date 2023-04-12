package main

import (
	"LemonCache"
	"flag"
	"fmt"
	"geecache"
	"log"
	"net/http"
)


var db = map[string]string{
	"张三":  "789",
	"李四": "456",
	"王五":  "123",
}

func createGroup() *LemonCache.Group {
	return LemonCache.NewGroup("scores",2<<10,LemonCache.GetterFunc(func(key string) ([]byte, error){
		log.Println("[SlowDB] search key",key)
		if v,ok := db[key]; ok{
			return []byte(v), nil
		}
		return nil,fmt.Errorf("%s not exist",key)
	}))
}

func startCacheServer(addr string,addrs []string,lemon *LemonCache.Group)  {
	peers := LemonCache.NewHTTPPool(addr)
	peers.Set(addrs...)
	lemon.RegisterPeers(peers)
	log.Println("lemoncache is running at",addr)
	//addr[7:]移除"http://" 前缀
	log.Fatal(http.ListenAndServe(addr[7:], peers))

}

func startAPIServer(apiAddr string, lemon *LemonCache.Group)  {
	http.Handle("/api",http.HandlerFunc(
		func(w http.ResponseWriter,r *http.Request) {
			key := r.URL.Query().Get("key")
			view,err := lemon.Get(key)
			if err != nil{
				http.Error(w,err.Error(),http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))

		log.Println("fontend server is running at",apiAddr)
		log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

//main() 函数需要命令行传入 port 和 api 2 个参数，用来在指定端口启动 HTTP 服务。
func main(){
	var port int
	var api bool
	flag.IntVar(&port,"port",8001,"LemonCache server port")
	flag.BoolVar(&api,"api",false,"Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	lemon := createGroup()
	if api {
		go startAPIServer(apiAddr,lemon)
	}
	startCacheServer(addrMap[port],addrs,lemon)
}