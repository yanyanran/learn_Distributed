package lru

import "container/list"

// Cache LRU缓存 并发访问不安全
type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存
	nbytes    int64                         // 当前已使用的内存
	ll        *list.List                    // 双向链表
	cache     map[string]*list.Element      // K：字符串 V：双向链表中对应节点的指针
	OnEvicted func(key string, value Value) // 某条记录被移除时的回调函数
}

type entry struct { // ll节点的数据类型
	key   string
	value Value
}

// Value 使用Len计算所需的字节数=> 通用
type Value interface {
	Len() int
}

// New Cache的构造函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 查找关键字的值
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok { // 从字典中找到对应的双向链表的节点
		c.ll.MoveToFront(ele) // 移动到队尾
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoverOldest 缓存淘汰 移除最近最少访问的节点（队首）
func (c *Cache) RemoverOldest() {
	ele := c.ll.Back() // 取队首delete
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                // 从字典c.cache中删除节点映射关系
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新当前所用内存
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok { // 键存在=> 更新对应节点的值，并将节点移到队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes { // 超过设定的最大值c.maxBytes
		c.RemoverOldest()
	}
}

// Len 缓存entry数
func (c *Cache) Len() int {
	return c.ll.Len()
}
