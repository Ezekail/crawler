package main

import (
	"fmt"
	"github.com/Ezekail/crawler.git/collect"
	"github.com/Ezekail/crawler.git/engine"
	"github.com/Ezekail/crawler.git/log"
	"github.com/Ezekail/crawler.git/parse/doubangroup"
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

	// 豆瓣cookie
	cookie := "bid=-UXUw--yL5g; dbcl2=\"214281202:q0BBm9YC2Yg\"; __yadk_uid=jigAbrEOKiwgbAaLUt0G3yPsvehXcvrs; push_noty_num=0; push_doumail_num=0; __utmz=30149280.1665849857.1.1.utmcsr=accounts.douban.com|utmccn=(referral)|utmcmd=referral|utmcct=/; __utmv=30149280.21428; ck=SAvm; _pk_ref.100001.8cb4=%5B%22%22%2C%22%22%2C1665925405%2C%22https%3A%2F%2Faccounts.douban.com%2F%22%5D; _pk_ses.100001.8cb4=*; __utma=30149280.2072705865.1665849857.1665849857.1665925407.2; __utmc=30149280; __utmt=1; __utmb=30149280.23.5.1665925419338; _pk_id.100001.8cb4=fc1581490bf2b70c.1665849856.2.1665925421.1665849856."
	// 使用了广度优先搜索算法。循环往复遍历 worklist 列表，完成爬取与解析的动作，找到所有符合条件的帖子。
	var seeds []*collect.Request
	for i := 0; i <= 100; i += 25 {
		str := fmt.Sprintf("https://www.douban.com/group/szsh/discussion?start=%d", i)
		seeds = append(seeds, &collect.Request{
			Url:       str,
			Cookie:    cookie,
			WaitTime:  1 * time.Second,
			ParseFunc: doubangroup.ParseURL,
		})
	}
	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}
	s := engine.ScheduleEngine{
		WorkCount: 5,
		Logger:    logger,
		Fetcher:   f,
		Seeds:     seeds,
	}
	s.Run()

}
