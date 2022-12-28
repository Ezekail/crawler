package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Task struct {
	Url         string // 要访问的网站
	Cookie      string
	WaitTime    time.Duration // 默认等待时间
	MaxDepth    int           // 最大爬取深度
	RootReq     *Request      // 任务中的第一个请求
	Reload      bool          // 网站是否可以重复爬取
	Fetcher     Fetcher
	Visited     map[string]bool
	VisitedLock sync.Mutex
}

type Request struct {
	unique    string
	Task      *Task
	Priority  int                                // 请求的优先级
	Url       string                             // 要访问的网站
	Method    string                             // 方法
	Depth     int                                // 任务的当前深度
	ParseFunc func([]byte, *Request) ParseResult // 解析从网站获取到的网站信息
}

type ParseResult struct {
	Requests []*Request    // 用于进一步获取数据
	Items    []interface{} // 获取到的数据
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}

// Unique 请求的唯一识别码
func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}
