package engine

import (
	"fmt"
	"github.com/Ezekail/crawler.git/collect"
	"go.uber.org/zap"
	"sync"
)

type Crawler struct {
	out         chan collect.ParseResult // 负责处理爬取后的数据
	Visited     map[string]bool          // 存储请求访问信息
	VisitedLock sync.Mutex               // 锁
	options
}

type Scheduler interface {
	Schedule()                // 启动调度器
	Push(...*collect.Request) // 将请求放入到调度器中
	Pull() *collect.Request   // 从调度器中获取请求
}

type Schedule struct {
	requestCh chan *collect.Request // 负责接收请求
	workCh    chan *collect.Request // 负责分配任务给 worker
	reqQueue  []*collect.Request
	Logger    *zap.Logger
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
		seed.RootReq.Task = seed
		seed.RootReq.Url = seed.Url
		reqs = append(reqs, seed.RootReq)
	}
	go c.scheduler.Schedule()
	go c.scheduler.Push(reqs...)
}

// Schedule 创建调度程序，接收任务并完成任务的调度
func (s *Schedule) Schedule() {
	for {
		var req *collect.Request
		var ch chan *collect.Request
		// 队列不为空，证明有爬虫任务
		if len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workCh
		}
		select {
		// 接收来自外界的请求，并将请求存储到 reqQueue 队列中
		case r := <-s.requestCh:
			s.reqQueue = append(s.reqQueue, r)
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
			continue
		}
		if err != nil {
			c.Logger.Error("can't fetch",
				zap.Error(err),
			)
			continue
		}
		// 解析服务器返回的数据
		result := r.ParseFunc(body, r)
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
