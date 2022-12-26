package proxy

import (
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
)

// 负责专门处理代理相关的操作

type ProxyFunc func(r *http.Request) (*url.URL, error)

type roundRobinSwitcher struct {
	proxyURLs []*url.URL
	index     uint32
}

// RoundRobinProxySwitcher 创建了一个代理切换器函数，该函数在每个请求上旋转ProxyURL。
// 代理类型由URL方案确定。支持“http”、“https”和“socks5”。如果方案为空，则假定为“http”。
func RoundRobinProxySwitcher(ProxyURLs ...string) (ProxyFunc, error) {
	// 接收代理服务器地址列表，将其字符串地址解析为 url.URL
	// 并放入到 roundRobinSwitcher 结构中，该结构中还包含了一个自增的序号 index
	if len(ProxyURLs) < 1 {
		return nil, errors.New("proxy URL list is empty")
	}
	urls := make([]*url.URL, len(ProxyURLs))
	for i, u := range ProxyURLs {
		parsedU, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		urls[i] = parsedU
	}
	return (&roundRobinSwitcher{urls, 0}).GetProxy, nil
}

// GetProxy 取余算法实现轮询调度
func (r *roundRobinSwitcher) GetProxy(pr *http.Request) (*url.URL, error) {
	index := atomic.AddUint32(&r.index, 1) - 1
	// 通过取余操作实现对代理地址的轮询
	u := r.proxyURLs[index%uint32(len(r.proxyURLs))]
	return u, nil
}
