package main

import (
	"bytes"
	"fmt"
	"github.com/Ezekail/crawler.git/collect"
	"github.com/PuerkitoBio/goquery"
	"strings"
	"time"
)

func main() {
	// url := "https://www.thepaper.cn/"
	// 需设置携带User-Agent访问才有数据
	url := "https://book.douban.com/subject/1007305/"
	//var f collect.Fetcher = &collect.BaseFetch{}
	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 300 * time.Millisecond,
	}

	body, err := f.Get(url)
	if err != nil {
		fmt.Printf("read content failed,%v", err)
		return
	}
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
	numLinks := strings.Count(string(body), "<a")
	fmt.Printf("homepage has %d links!\n", numLinks)

	exist := strings.Contains(string(body), "疫情")
	fmt.Printf("是否存在疫情:%v\n", exist)
}
