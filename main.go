package main

import (
	"github.com/Ezekail/crawler.git/collect"
	"github.com/Ezekail/crawler.git/engine"
	"github.com/Ezekail/crawler.git/log"
	"github.com/Ezekail/crawler.git/proxy"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	// log
	plugin := log.NewStdoutPlugin(zapcore.InfoLevel)
	//plugin, closer := log.NewFilePlugin("./log.txt", zapcore.InfoLevel)
	//defer closer.Close()
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// 开启两个代理地址
	proxyURLs := []string{"http://127.0.0.1:8888", "http://127.0.0.1:8888"}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")
	}

	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}
	var seeds = make([]*collect.Task, 0, 1000)

	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "js_find_douban_sun_room",
		},
		Fetcher: f,
	})
	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	s.Run()

}
