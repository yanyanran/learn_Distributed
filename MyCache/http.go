package MyCache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// 提供被其他节点访问的能力(基于http)

type HTTPPool struct {
	self     string // 自己的地址(主机名/IP和端口
	basePath string // 作节点间通讯地址的前缀
}

const defaultBasePath = "/_mycache/"

// NewHTTPPool 初始化节点的HTTP池
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 带服务器名称的日志
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 处理所有HTTP请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) { // 判断访问路径前缀是否是basePath
		panic("HTTPPool提供意外路径：" + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// 约定访问路径格式为/<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "错误的请求", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName) // 通过groupname得到group实例
	if group == nil {
		http.Error(w, "没有这样的group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key) // Get获取缓存数据
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice()) // 将缓存值作为httpResponse的body返回
}
