package geecache

import (
	"fmt"
	"geecache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// HTTP缓存池
type HTTPPool struct {
	// 用来记录自己的地址，包括主机名/IP 和端口。
	self string
	// 作为节点间通讯地址的前缀，默认是 /_geecache/
	basePath string
	mu       sync.Mutex // guards peers and httpGetters
	// 新增成员变量 peers，类型是一致性哈希算法的 Map，用来根据具体的 key 选择节点。
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

type httpGetter struct {
	baseURL string
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 如果请求路径不以‘basePath’开头，将Panic
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	// 记录请求方法和路径
	p.Log("%s %s", r.Method, r.URL.Path)

	// 从请求路径中解析出组名（groupName）和键名（key）。
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)

	// 如果解析后的路径部分数量不为2，返回"bad request"和HTTP状态码400。
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no suck group: "+groupName, http.StatusNotFound)
		return
	}

	// 通过组的Get方法获取缓存项（view），如果获取失败则返回错误信息和HTTP状态码500。
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置响应头的"Content-Type"为"application/octet-stream"，表示响应内容是二进制流。
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// 实例化了一致性哈希算法，并且添加了传入的节点。并为每一个节点创建了一个 HTTP 客户端 httpGetter。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
// PickPeer picks a peer according to key
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// var _ PeerGetter = (*httpGetter)(nil) 这行代码实际上是在静态检查编译时确认 httpGetter 类型是否实现了 PeerGetter 接口。如果 httpGetter 类型没有实现 PeerGetter 接口，编译器会在编译时报错。
// 如果 httpGetter 类型实现了 PeerGetter 接口，这个声明将通过编译，否则会导致编译错误。
var _ PeerGetter = (*httpGetter)(nil)
