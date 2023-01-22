package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash    Hash           // 可自定义哈希函数
	n       int            // 虚拟节点倍数
	keys    []int          // 哈希环（已排序
	hashMap map[int]string // 虚拟节点与真实节点的映射表（K：虚拟节点的哈希值 V：真实节点的名称
}

func New(n int, fn Hash) *Map {
	m := &Map{
		n:       n,
		hash:    fn,
		hashMap: make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE // 默认哈希算法
	}
	return m
}

// Add 实现添加真实节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.n; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash) // 加到环上
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys) // 环上的哈希值排序
}

func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool { // 匹配的虚拟节点下标
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]] // 通过hashMap映射得到真实的节点
}
