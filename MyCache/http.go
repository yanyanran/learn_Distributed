package MyCache

import (
	"MyCache/consistenthash"
	pb "MyCache/mycachepb"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 提供被其他节点访问的能力(基于http)

type HTTPPool struct {
	self        string                 // 自己的地址(主机名/IP和端口
	basePath    string                 // 作节点间通讯地址的前缀
	mu          sync.Mutex             // 保护peers和httpGetter
	peers       *consistenthash.Map    // 根据具体key选择节点
	httpGetters map[string]*httpGetter // 映射远程节点与对应的httpGetter
}

const (
	defaultBasePath = "/_mycache/"
	defaultReplicas = 50
)

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
	// 将value作为proto消息写入响应body
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()}) // 编码HTTP响应
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body) // 将缓存值作为httpResponse的body返回
}

// HTTP客户端
type httpGetter struct {
	baseURL string // 远程节点地址
}

var _ PeerGetter = (*httpGetter)(nil)

// Get 实现PeerGetter接口=> 取返回值并转为[]bytes型
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	res, err := http.Get(fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()), // 转义字符串以便安全放在URL路径段中
		url.QueryEscape(in.GetKey()),
	))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body) // 从io.Reader中读数据直到结尾
	if err != nil {
		return fmt.Errorf("读取响应body: %v", err)
	}
	if err = proto.Unmarshal(bytes, out); err != nil { // 解码HTTP响应
		return fmt.Errorf("解码响应body: %v", err)
	}
	return nil
}

// Set 更新连接池peer列表
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 实例化一致性哈希算法
	p.peers = consistenthash.New(defaultReplicas, nil)
	// 添加传入节点
	p.peers.Add(peers...)

	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers { // 为每个节点创建HTTP客户端（？
		p.httpGetters[peer] = &httpGetter{
			baseURL: peer + p.basePath,
		}
	}
}

var _ PeerPicker = (*HTTPPool)(nil)

// PickPeer 根据key选择peer
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 一致性哈希算法的Get()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("选择peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}
