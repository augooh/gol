package lru

import "container/list"

// lru Cache, 并发访问不安全
type Cache struct {
	maxBytes int64
	nbytes   int64
	// Go 语言标准库实现的双向链表list.List
	ll    *list.List
	cache map[string]*list.Element
	// OnEvicted 是一个回调函数，允许用户在缓存淘汰条目时执行自定义的逻辑。
	// 当缓存中的某个键值对因为LRU（Least Recently Used，最近最少使用）策略被移除时，OnEvicted 函数会被调用，并传递被淘汰的键和值作为参数。
	// 用户可以通过设置 OnEvicted 字段为自己的函数来定义在缓存淘汰时应该执行的操作，例如释放资源、记录日志等。
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

func (c *Cache) RemoveOldest() {
	// 取到队首节点，从链表中删除
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从字典中 c.cache 删除该节点的映射关系
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	// key存在，直接更新对应节点的值，并将节点移到最尾
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// 不存在的话添加新节点
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
