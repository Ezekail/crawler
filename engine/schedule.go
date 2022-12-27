package engine

import (
	"github.com/Ezekail/crawler.git/collect"
	"go.uber.org/zap"
)

type Schedule struct {
	requestCh chan *collect.Request    // 负责接收请求
	workCh    chan *collect.Request    // 负责分配任务给 worker
	out       chan collect.ParseResult // 负责处理爬取后的数据
	options
}

type Config struct {
	WorkCount int // 执行任务的数量
	Fetcher   collect.Fetcher
	Logger    *zap.Logger
	Seeds     []*collect.Request
}

func NewSchedule(opts ...Option) *Schedule {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	s := &Schedule{}
	s.options = options
	return s
}

func (s *Schedule) Run() {
	requestCh := make(chan *collect.Request)
	workCh := make(chan *collect.Request)
	out := make(chan collect.ParseResult)
	s.requestCh = requestCh
	s.workCh = workCh
	s.out = out
	go s.Schedule()
	// 创建指定数量的 worker，完成实际任务的处理
	for i := 0; i < s.WorkCount; i++ {
		go s.CreateWork()
	}
	s.HandleResult()
}

// Schedule 创建调度程序，接收任务并完成任务的调度
func (s *Schedule) Schedule() {
	var reqQueue = s.Seeds
	for {
		var req *collect.Request
		var ch chan *collect.Request
		// 队列不为空，证明有爬虫任务
		if len(reqQueue) > 0 {
			req = reqQueue[0]
			reqQueue = reqQueue[1:]
			ch = s.workCh
		}
		select {
		// 接收来自外界的请求，并将请求存储到 reqQueue 队列中
		case r := <-s.requestCh:
			reqQueue = append(reqQueue, r)
		// 将任务发送到 workerCh 通道中，等待 worker 接收
		case ch <- req:
		}
	}
}

// CreateWork 创建指定数量的 worker，完成实际任务的处理
func (s *Schedule) CreateWork() {
	for {
		//接收到调度器分配的任务
		r := <-s.workCh
		// 访问服务器
		body, err := s.Fetcher.Get(r)
		if err != nil {
			s.Logger.Error("can't fetch",
				zap.Error(err),
			)
			continue
		}
		// 解析服务器返回的数据
		result := r.ParseFunc(body, r)
		// 将返回的数据发送到 out 通道中，方便后续的处理
		s.out <- result
	}
}

// HandleResult 创建数据处理协程，对爬取到的数据进行进一步处理
func (s Schedule) HandleResult() {
	for {
		select {
		case result := <-s.out:
			for _, req := range result.Requests {
				s.requestCh <- req
			}
			for _, item := range result.Items {
				// Todo : 存储
				s.Logger.Sugar().Info("get result", item)
			}
		}
	}
}
