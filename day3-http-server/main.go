package main

import (
	"LemonCache"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var db = map[string]string{
	"张三":  "789",
	"李四": "456",
	"王五":  "123",
}

func main(){
	url := "./img1.png"
	// LemonCache.NewGroup("scores",2<<10,LemonCache.GetterFunc(func(key string) ([]byte, error){
	// 	log.Println("[SlowDB] search key",key)
	// 	if v,ok := db[key];ok{
	// 		return []byte(v),nil
	// 	}
	// 	return nil,fmt.Errorf("%s not exist",key)
	// }))
	LemonCache.NewGroup("img",2<<20,LemonCache.GetterFunc(func(key string) ([]byte,error){
		log.Println("[Local] search key",key)
		if key == "img1.png"{
			v,err := getLocal(url); 
			if err == nil{
				return v,nil
			}
		}
		return nil,fmt.Errorf("%s not exist",key)
	}))

	addr := "localhost:9999"
	peers := LemonCache.NewHTTPPool(addr)
	log.Println("lemoncache is running at",addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}

func getLocal(url string) ([]byte, error) {
	fp, err := os.OpenFile(url, os.O_CREATE|os.O_APPEND, 6) // 读写方式打开
	if err != nil {
		// 如果有错误返回错误内容
		return nil, err
	}
	defer fp.Close()
	bytes, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, err
	}
	return bytes, err
}