package MyCache

// PeerPicker 用于定位拥有特定密钥的peer
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool) // 根据传入的key选择相应节点PeerGetter
}

// PeerGetter  HTTP客户端接口 必须由peer实现
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
