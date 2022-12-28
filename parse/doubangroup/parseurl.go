package doubangroup

import (
	"github.com/Ezekail/crawler.git/collect"
	"regexp"
)

const urlListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`

// ParseURL 获取所有帖子的 URL
func ParseURL(contents []byte, req *collect.Request) collect.ParseResult {
	re := regexp.MustCompile(urlListRe)

	matches := re.FindAllSubmatch(contents, -1)
	result := collect.ParseResult{}
	//匹配到符合帖子格式的 URL 我们把它组装到一个新的 Request 中，用作下一步的爬取
	for _, m := range matches {
		u := string(m[1])
		// 在添加下一层的 URL 时，我们将 Depth 加 1
		result.Requests = append(result.Requests, &collect.Request{
			Method: "GET",
			Url:    u,
			Task:   req.Task,
			Depth:  req.Depth + 1,
			ParseFunc: func(c []byte, request *collect.Request) collect.ParseResult {
				return GetContent(c, u)
			},
		})
	}
	return result
}

const ContentRe = `<div class="topic-content">[\s\S]*?阳台[\s\S]*?<div`

// 想要获取的是正文中带有“阳台”字样的帖子

func GetContent(contents []byte, url string) collect.ParseResult {
	re := regexp.MustCompile(ContentRe)
	ok := re.Match(contents)
	if !ok {
		return collect.ParseResult{
			Items: []interface{}{},
		}
	}
	// 匹配则把URL加入结果集返回
	result := collect.ParseResult{
		Items: []interface{}{url},
	}
	return result
}
