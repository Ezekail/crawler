package collect

import (
	"errors"
	"time"
)

type Request struct {
	Url       string // 要访问的网站
	Cookie    string
	WaitTime  time.Duration
	Depth     int                                // 任务的当前深度
	MaxDepth  int                                // 最大爬取深度
	ParseFunc func([]byte, *Request) ParseResult // 解析从网站获取到的网站信息
}

type ParseResult struct {
	Requests []*Request    // 用于进一步获取数据
	Items    []interface{} // 获取到的数据
}

func (r *Request) Check() error {
	if r.Depth > r.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}
