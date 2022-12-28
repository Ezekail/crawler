package engine

import (
	"fmt"
	"github.com/Ezekail/crawler.git/collect"
	"github.com/Ezekail/crawler.git/parse/doubangroup"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"sync"
)

// 初始化任务与规则
func init() {
	Store.Add(doubangroup.DoubangroupTask)
	Store.AddJSTask(doubangroup.DoubangroupJSTask)
}

func (c *CrawlerStore) Add(task *collect.Task) {
	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

type mystruct struct {
	Name string
	Age  int
}

// AddJsReqs 用于动态规则添加请求。
func AddJsReqs(jreqs []map[string]interface{}) []*collect.Request {
	// 将在 JS 脚本中的请求数据变为 Go 结构中的数组[]*collect.Request
	reqs := make([]*collect.Request, 0)

	for _, jreq := range jreqs {
		req := &collect.Request{}
		u, ok := jreq["Url"].(string)
		if !ok {
			return nil
		}
		req.Url = u
		req.RuleName, _ = jreq["RuleName"].(string)
		req.Method, _ = jreq["Method"].(string)
		req.Priority, _ = jreq["Priority"].(int)
		reqs = append(reqs, req)
	}
	return reqs
}

// AddJsReq 用于动态规则添加单个请求。
func AddJsReq(jreq map[string]interface{}) []*collect.Request {
	reqs := make([]*collect.Request, 0)
	req := &collect.Request{}
	u, ok := jreq["Url"].(string)
	if !ok {
		return nil
	}
	req.Url = u
	req.RuleName, _ = jreq["RuleName"].(string)
	req.Method, _ = jreq["Method"].(string)
	req.Priority, _ = jreq["Priority"].(int)
	reqs = append(reqs, req)
	return reqs
}

func (c *CrawlerStore) AddJSTask(m *collect.TaskModel) {
	task := &collect.Task{
		Property: m.Property,
	}

	task.Rule.Root = func() ([]*collect.Request, error) {
		vm := otto.New()
		vm.Set("AddJsReq", AddJsReqs)
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}
		return e.([]*collect.Request), nil
	}

	for _, r := range m.Rules {
		paesrFunc := func(parse string) func(ctx *collect.Context) (collect.ParseResult, error) {
			return func(ctx *collect.Context) (collect.ParseResult, error) {
				vm := otto.New()
				vm.Set("ctx", ctx)
				v, err := vm.Eval(parse)
				if err != nil {
					return collect.ParseResult{}, err
				}
				e, err := v.Export()
				if err != nil {
					return collect.ParseResult{}, err
				}
				if e == nil {
					return collect.ParseResult{}, err
				}
				return e.(collect.ParseResult), err
			}
		}(r.ParseFunc)
		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*collect.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &collect.Rule{
			paesrFunc,
		}
	}

	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

// Store 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*collect.Task{},
	hash: map[string]*collect.Task{},
}

type CrawlerStore struct {
	list []*collect.Task
	hash map[string]*collect.Task
}

type Crawler struct {
	out         chan collect.ParseResult // 负责处理爬取后的数据
	Visited     map[string]bool          // 存储请求访问信息
	VisitedLock sync.Mutex               // 锁

	failures    map[string]*collect.Request // 失败请求id -> 失败请求
	failureLock sync.Mutex
	options
}

type Scheduler interface {
	Schedule()                // 启动调度器
	Push(...*collect.Request) // 将请求放入到调度器中
	Pull() *collect.Request   // 从调度器中获取请求
}

type Schedule struct {
	requestCh   chan *collect.Request // 负责接收请求
	workCh      chan *collect.Request // 负责分配任务给 worker
	priReqQueue []*collect.Request    // 优先队列
	reqQueue    []*collect.Request    // 普通队列
	Logger      *zap.Logger
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	c := &Crawler{}
	out := make(chan collect.ParseResult)
	c.out = out
	c.options = options
	return c
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	requestCh := make(chan *collect.Request)
	workCh := make(chan *collect.Request)
	s.requestCh = requestCh
	s.workCh = workCh
	return s
}

func (c *Crawler) Run() {
	go c.Schedule()
	// 创建指定数量的 worker，完成实际任务的处理
	for i := 0; i < c.WorkCount; i++ {
		go c.CreateWork()
	}
	c.HandleResult()
}

func (s *Schedule) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

func (s *Schedule) Pull() *collect.Request {
	r := <-s.workCh
	return r
}
func (s *Schedule) Output() *collect.Request {
	r := <-s.workCh
	return r
}

func (c *Crawler) Schedule() {
	var reqs []*collect.Request
	for _, seed := range c.Seeds {
		task := Store.hash[seed.Name]
		task.Fetcher = seed.Fetcher
		// 获取初始化任务
		rootReqs, err := task.Rule.Root()
		if err != nil {
			c.Logger.Error("get root failed",
				zap.Error(err),
			)
			continue
		}
		for _, req := range rootReqs {
			req.Task = task
		}
		reqs = append(reqs, rootReqs...)
	}
	go c.scheduler.Schedule()
	go c.scheduler.Push(reqs...)
}

// Schedule 创建调度程序，接收任务并完成任务的调度
func (s *Schedule) Schedule() {
	var req *collect.Request
	var ch chan *collect.Request
	for {
		// 优先从优先队列中获取请求
		if req == nil && len(s.priReqQueue) > 0 {
			req = s.priReqQueue[0]
			s.priReqQueue = s.priReqQueue[1:]
			ch = s.workCh
		}
		// 队列不为空，证明有爬虫任务
		if req == nil && len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workCh
		}
		select {
		// 接收来自外界的请求，并将请求存储到 reqQueue 队列中
		// 请求的优先级更高，也会单独放入优先级队列
		case r := <-s.requestCh:
			if r.Priority > 0 {
				s.priReqQueue = append(s.priReqQueue, r)
			} else {
				s.reqQueue = append(s.reqQueue, r)
			}
		// 将任务发送到 workerCh 通道中，等待 worker 接收
		case ch <- req:
			fmt.Println(123)
		}
	}
}

// CreateWork 创建指定数量的 worker，完成实际任务的处理
func (c *Crawler) CreateWork() {
	for {
		//接收到调度器分配的任务
		r := c.scheduler.Pull()
		// 检查是否超过最大爬取深度
		if err := r.Check(); err != nil {
			c.Logger.Error("check failed",
				zap.Error(err),
			)
			continue
		}
		// 访问服务器
		body, err := r.Task.Fetcher.Get(r)
		if len(body) < 6000 {
			c.Logger.Error("can't fetch ",
				zap.Int("length", len(body)),
				zap.String("url", r.Url),
			)
			c.SetFailure(r)
			continue
		}
		if err != nil {
			c.Logger.Error("can't fetch",
				zap.Error(err),
			)
			c.SetFailure(r)
			continue
		}
		// 解析服务器返回的数据
		//获取当前任务对应的规则
		rule := r.Task.Rule.Trunk[r.RuleName]
		// 内容解析
		result, err := rule.ParseFunc(&collect.Context{
			Body: body,
			Req:  r,
		})
		if err != nil {
			c.Logger.Error("ParseFunc failed ",
				zap.Error(err),
				zap.String("url", r.Url),
			)
			continue
		}
		// 新的任务加入队列中
		// 将返回的数据发送到 out 通道中，方便后续的处理
		if len(result.Requests) > 0 {
			go c.scheduler.Push(result.Requests...)
		}
		c.out <- result
	}
}

// HandleResult 创建数据处理协程，对爬取到的数据进行进一步处理
func (c *Crawler) HandleResult() {
	for {
		select {
		case result := <-c.out:
			for _, item := range result.Items {
				// Todo : 存储
				c.Logger.Sugar().Info("get result", item)
			}
		}
	}
}

func (c *Crawler) HasVisited(r *collect.Request) bool {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()
	unique := r.Unique()
	return c.Visited[unique]
}

func (c *Crawler) StoreVisited(reqs ...*collect.Request) {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()
	for _, req := range reqs {
		unique := req.Unique()
		c.Visited[unique] = true
	}
}

// SetFailure 当请求失败之后，将请求加入到 failures 哈希表中，并且把它重新交由调度引擎进行调度
func (c *Crawler) SetFailure(req *collect.Request) {
	//如果不可以重复爬取，我们需要在失败重试前删除 Visited 中的历史记录
	if !req.Task.Reload {
		c.VisitedLock.Lock()
		unique := req.Unique()
		delete(c.Visited, unique)
		c.VisitedLock.Unlock()
	}
	c.failureLock.Lock()
	defer c.failureLock.Unlock()
	if _, ok := c.failures[req.Unique()]; !ok {
		// 首次失败时，再重新执行一次
		c.failures[req.Unique()] = req
		c.scheduler.Push(req)
	}
	//todo: 失败2次，加载到失败队列中
}
