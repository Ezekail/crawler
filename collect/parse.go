package collect

// RuleTree 采集规则树
type RuleTree struct {
	Root  func() ([]*Request, error) // 根节点(执行入口)，用于生成爬虫的种子网站
	Trunk map[string]*Rule           // 规则哈希表，存储当前任务的所有规则
}

// Rule 采集规则节点
type Rule struct {
	ParseFunc func(*Context) (ParseResult, error) // 内容解析函数
}
