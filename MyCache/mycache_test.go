package MyCache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db)) // loadCounts统计某个key调用回调函数的次数
	myGroup := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB]搜索key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := myGroup.Get(k); err != nil || view.String() != v {
			t.Fatalf("无法获取%s的值", k)
		} // 缓存为空情况下通过回调函数获取源数据
		if _, err := myGroup.Get(k); err != nil || loadCounts[k] > 1 { // loadCounts>1表示调用了多次回调函数，没有缓存
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := myGroup.Get("unknown"); err == nil {
		t.Fatalf("unknown的value应为空，但得到了%s", view)
	}
}
