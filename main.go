package main

import (
	"bufio"
	"fmt"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// 前解析好正则表达式内容
var headerRe = regexp.MustCompile(`<div class="small_cardcontent__BTALp"[\s\S]*?<h2>([\s\S]*?)</h2>`)

func main() {
	url := "https://www.thepaper.cn/"
	body, err := Fetch(url)
	if err != nil {
		log.Printf("read content failed,%v", err)
	}
	matches := headerRe.FindAllSubmatch(body, -1)
	for _, match := range matches {
		fmt.Println("fetch card news:", string(match[1]))
	}
	numLinks := strings.Count(string(body), "<a")
	fmt.Printf("homepage has %d links!\n", numLinks)

	exist := strings.Contains(string(body), "疫情")
	fmt.Printf("是否存在疫情:%v\n", exist)
}

// Fetch 获取网页的内容，检测网页的字符编码并将文本统一转换为 UTF-8 格式
func Fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error status code:%v", resp.StatusCode)
	}
	bodyReader := bufio.NewReader(resp.Body)
	e := DeterMinEncoding(bodyReader)
	// transform.NewReader 用于将 HTML 文本从特定编码转换为 UTF-8 编码
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return ioutil.ReadAll(utf8Reader)
}

// DeterMinEncoding 检测并返回当前 HTML 文本的编码格式
func DeterMinEncoding(r *bufio.Reader) encoding.Encoding {
	//如果返回的 HTML 文本小于 1024 字节，我们认为当前 HTML 文本有问题，直接返回默认的 UTF-8 编码就好了
	bytes, err := r.Peek(1024)
	if err != nil {
		log.Printf("fetch error:%v", err)
		return unicode.UTF8
	}
	// 检测并返回对应 HTML 文本的编码
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}
