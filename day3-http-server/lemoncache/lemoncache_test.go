package LemonCache

import (
	"reflect"
	"testing"
	"fmt"
	"log"
)

var db = map[string]string{
	"张三":  "789",
	"李四": "456",
	"王五":  "123",
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte,error) {
		return []byte(key),nil
	})

	expect := []byte("key")
	if v,_ := f.Get("key"); !reflect.DeepEqual(v,expect){
		t.Errorf("callback failed")
	}
}

//创建 group 实例，并测试 Get 方法
func TestGet(t *testing.T) {
	loadCounts := make(map[string]int,len(db))
	lemon := NewGroup("scores",2<<10,GetterFunc(func(key string) ([]byte,error) {
		log.Println("[SlowDB] search key",key)
		if v,ok := db[key];ok{
			if _,ok := loadCounts[key];!ok{
				loadCounts[key] = 0
			}
			loadCounts[key]++
			return []byte(v),nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))

	for k,v := range db{
		if view,err := lemon.Get(k); err != nil || view.String() != v{
			t.Fatal("failed to get value of 张三")
		} // 从回调函数中获取
		if _, err := lemon.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // 缓存命中
	}

	if view,err := lemon.Get("unkown"); err == nil{
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
