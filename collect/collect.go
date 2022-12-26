package collect

import (
	"bufio"
	"fmt"
	"github.com/Ezekail/crawler.git/proxy"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"net/http"
	"time"
)

type Fetcher interface {
	Get(url string) ([]byte, error)
}

type BaseFetch struct {
}

// Get 获取网页的内容，检测网页的字符编码并将文本统一转换为 UTF-8 格式
func (b *BaseFetch) Get(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error status code:%d\n", resp.StatusCode)
		return nil, err
	}
	bodyReader := bufio.NewReader(resp.Body)
	e := DeterMinEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return ioutil.ReadAll(utf8Reader)
}

type BrowserFetch struct {
	Timeout time.Duration
	Proxy   proxy.ProxyFunc
}

// Get 模拟浏览器访问
func (b BrowserFetch) Get(url string) ([]byte, error) {
	// 创建一个 HTTP 客户端 http.Client
	client := &http.Client{
		Timeout: b.Timeout,
	}
	// 更新 http.Client 变量中的 Transport 结构中的 Proxy 函数，将其替换为我们自定义的代理函数
	if b.Proxy != nil {
		transport := http.DefaultTransport.(*http.Transport)
		transport.Proxy = b.Proxy
		client.Transport = transport
	}
	// 然后通过 http.NewRequest 创建一个请求
	request, err := http.NewRequest("Get", url, nil)
	if err != nil {
		return nil, fmt.Errorf("get url failed,err:%v\n", err)
	}
	// 在请求中调用 req.Header.Set 设置 User-Agent 请求头
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	// 最后调用 client.Do 完成 HTTP 请求
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	bodyReader := bufio.NewReader(response.Body)
	e := DeterMinEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return ioutil.ReadAll(utf8Reader)
}

// DeterMinEncoding 检测并返回当前 HTML 文本的编码格式
func DeterMinEncoding(r *bufio.Reader) encoding.Encoding {
	//如果返回的 HTML 文本小于 1024 字节，我们认为当前 HTML 文本有问题，直接返回默认的 UTF-8 编码就好了
	bytes, err := r.Peek(1024)
	if err != nil {
		fmt.Printf("fetch error:%v", err)
		return unicode.UTF8
	}
	// 检测并返回对应 HTML 文本的编码
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}
