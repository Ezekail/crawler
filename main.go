package main

import (
	"bytes"
	"fmt"
	"github.com/Ezekail/crawler.git/collect"
	"github.com/Ezekail/crawler.git/log"
	"github.com/Ezekail/crawler.git/proxy"
	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	plugin, closer := log.NewFilePlugin("./log.txt", zapcore.InfoLevel)
	defer closer.Close()
	logger := log.NewLogger(plugin)
	logger.Info("log init end")
	// 开启两个代理地址
	proxyURLs := []string{"http://127.0.0.1:8888", "http://127.0.0.1:8888"}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")
	}
	// 开启两个代理地址访问到谷歌网站
	url := "https://google.com"
	//var f collect.Fetcher = &collect.BaseFetch{}
	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Proxy:   p,
	}

	body, err := f.Get(url)
	if err != nil {
		logger.Error("read content failed,%v", zap.Error(err))
		return
	}
	logger.Info("get content", zap.Int("len", len(body)))
	fmt.Println(string(body))

	// Css选择器匹配提取
	// 加载HTML文档
	cssDoc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		fmt.Println("read content failed,err:%v", err)
	}
	cssDoc.Find("div.small_cardcontent__BTALp h2").Each(func(i int, selection *goquery.Selection) {
		// 获取匹配标签中的文本
		title := selection.Text()
		fmt.Printf("Review %d: %s\n", i, title)
	})

}
