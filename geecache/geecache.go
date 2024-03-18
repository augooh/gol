package geecache

// 远程节点（Remote Node）：
// 远程节点通常是指分布式缓存系统中的其他节点或者其他服务，它们可能位于不同的物理服务器或者逻辑服务器上。
// 远程节点存储着系统中的部分数据，通常用于实现数据的分布式存储和缓存。
// 在分布式缓存系统中，客户端可以向远程节点发送请求，从远程节点获取数据。

// 缓存（Cache）：
// 缓存是指存储数据的临时存储介质，通常位于内存或者其他高速存储介质中。
// 在分布式缓存系统中，缓存通常是指每个节点上的本地缓存，用于存储该节点所负责的部分数据，以加速数据的访问。
// 缓存可以存储从远程节点获取的数据，也可以存储通过其他方式获取的数据。

// 本地缓存（Local Cache）：
// 本地缓存是指应用程序内部的缓存，通常位于内存中，用于临时存储应用程序需要频繁访问的数据，以加速数据的访问。
// 本地缓存通常用于缓存经常被访问的数据，例如，数据库查询结果、网络请求的响应等。
// 在分布式缓存系统中，每个节点通常会维护一个本地缓存，用于存储从远程节点获取的数据，以减少对远程节点的访问。

import (
	"fmt"
	"geecache/singleflight"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get value for a key from cache
// 在缓存中找数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

// 将 getLocally 封装在 load 方法中也可以使得后续对获取数据的逻辑进行修改或者扩展更加方便。
// 如果未来需要实现一些额外的逻辑，比如数据的预加载、数据的异步加载等，只需要在 load 方法中进行相应的修改即可
// func (g *Group) load(key string) (value ByteView, err error) {
// 	return g.getLocally(key)
// }

// 它首先检查是否已经注册了 PeerPicker，如果有注册，它会调用 PeerPicker 来选择一个远程节点，然后调用 getFromPeer 方法从选定的远程节点获取数据。
// 如果获取成功，则返回获取到的数据；如果获取失败，则尝试从本地缓存中获取数据。如果未注册
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// 找不到的话调用load-再调用getLocally
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	// 将这个值添加到缓存中
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
