package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"regexp"
	"sync"
	"time"
)

type Property struct {
	Name     string        `json:"name"` // 用户界面显示的名称（应保证唯一性）
	Url      string        `json:"url"`  // 要访问的网站
	Cookie   string        `json:"cookie"`
	WaitTime time.Duration `json:"wait_time"` // 默认等待时间
	Reload   bool          `json:"reload"`    // 网站是否可以重复爬取
	MaxDepth int           `json:"max_depth"` // 最大爬取深度
}

// Task 一个任务实例
type Task struct {
	Property
	Fetcher     Fetcher
	Rule        RuleTree
	Visited     map[string]bool
	VisitedLock sync.Mutex
}

// Context 用于传递上下文信息
type Context struct {
	Body []byte   // 要解析的内容字节数组
	Req  *Request // 当前的请求参数
}

// ParseJSReg 动态解析 JS 中传递的正则表达式并生成新的请求
func (c *Context) ParseJSReg(name string, reg string) ParseResult {
	re := regexp.MustCompile(reg)

	matches := re.FindAllSubmatch(c.Body, -1)
	result := ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(
			result.Requests, &Request{
				Method:   "GET",
				Task:     c.Req.Task,
				Url:      u,
				Depth:    c.Req.Depth + 1,
				RuleName: name,
			})
	}
	return result
}

// OutputJS 负责解析传递过来的正则表达式并完成结果的输出
func (c *Context) OutputJS(reg string) ParseResult {
	re := regexp.MustCompile(reg)
	ok := re.Match(c.Body)
	if !ok {
		return ParseResult{
			Items: []interface{}{},
		}
	}
	result := ParseResult{
		Items: []interface{}{c.Req.Url},
	}
	return result
}

// Request 单个请求
type Request struct {
	unique   string
	Task     *Task
	Priority int    // 请求的优先级
	Url      string // 要访问的网站
	Method   string // 方法
	Depth    int    // 任务的当前深度
	RuleName string
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
