package collect

type Request struct {
	Url       string // 要访问的网站
	Cookie    string
	ParseFunc func([]byte, *Request) ParseResult // 解析从网站获取到的网站信息
}

type ParseResult struct {
	Requests []*Request    // 用于进一步获取数据
	Items    []interface{} // 获取到的数据
}
