package collect

type (
	// TaskModel 动态规则模型
	TaskModel struct {
		Property
		Root  string      `json:"root_script"` // 初始化种子节点的 JS 脚本
		Rules []RuleModel `json:"rule"`        // 具体爬虫任务的规则树。
	}
	RuleModel struct {
		Name      string `json:"name"`
		ParseFunc string `json:"parse_script"`
	}
)
