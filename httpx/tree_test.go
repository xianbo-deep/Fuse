package httpx

import (
	"testing"
)

// TestNode_Insert 测试路由树的插入逻辑，包括正常插入和冲突检测
func TestNode_Insert(t *testing.T) {
	root := &node{}

	tests := []struct {
		description string
		route       string
		wantErr     bool
	}{
		{"Static route", "/", false},
		{"Static route nested", "/hello", false},
		{"Param route", "/user/:id", false},
		{"Conflict param", "/user/:name", true}, // 冲突: :id 已存在
		{"Conflict mixed", "/user/profile", false},
		{"Wildcard route", "/assets/*filepath", false},
		{"Conflict wildcard", "/assets/:id", true}, // 冲突: *filepath 已存在
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := root.insert(tt.route, tt.route)
			if (err != nil) != tt.wantErr {
				t.Errorf("insert(%q) error = %v, wantErr %v", tt.route, err, tt.wantErr)
			}
		})
	}
}

// TestNode_Search 测试路由查找匹配及参数提取
func TestNode_Search(t *testing.T) {
	root := &node{}
	// 初始化一批路由
	routes := []string{
		"/",
		"/hello",
		"/user/:id",
		"/user/:id/details",
		"/src/*filepath",
		"/ab:cd", // 静态路由即使包含冒号（非开头）也应处理为静态
	}

	for _, r := range routes {
		if err := root.insert(r, r); err != nil {
			t.Fatalf("Setup failed, insert %s error: %v", r, err)
		}
	}

	tests := []struct {
		path       string
		wantRoute  string
		wantParams map[string]string
		found      bool
	}{
		// 静态匹配
		{"/", "/", nil, true},
		{"/hello", "/hello", nil, true},

		// 参数匹配
		{"/user/42", "/user/:id", map[string]string{"id": "42"}, true},
		{"/user/alice", "/user/:id", map[string]string{"id": "alice"}, true},
		{"/user/42/details", "/user/:id/details", map[string]string{"id": "42"}, true},

		// 通配符匹配
		{"/src/main.go", "/src/*filepath", map[string]string{"filepath": "main.go"}, true},
		{"/src/cmd/app/main.go", "/src/*filepath", map[string]string{"filepath": "cmd/app/main.go"}, true},

		// 边缘情况
		{"/ab:cd", "/ab:cd", nil, true},   // 精确匹配
		{"/unknown", "", nil, false},      // 未找到
		{"/user/42/oops", "", nil, false}, // 子路径未匹配
		{"/user/alice/details", "/user/:id/details", map[string]string{"id": "alice"}, true}, // 新增测试用例
		{"/src//main.go", "/src/*filepath", map[string]string{"filepath": "/main.go"}, true}, // 新增测试用例，验证双斜杠情况
	}

	for _, tt := range tests {
		t.Run("Match: "+tt.path, func(t *testing.T) {
			params := make(map[string]string)
			n := root.search(tt.path, params)

			if !tt.found {
				if n != nil {
					t.Errorf("search(%q) = %v, want nil", tt.path, n)
				}
				return
			}

			if n == nil {
				t.Fatalf("search(%q) = nil, want %q", tt.path, tt.wantRoute)
			}

			if n.pattern != tt.wantRoute {
				t.Errorf("search(%q) pattern = %q, want %q", tt.path, n.pattern, tt.wantRoute)
			}

			if len(tt.wantParams) > 0 {
				for k, v := range tt.wantParams {
					if params[k] != v {
						t.Errorf("search(%q) param %q = %q, want %q", tt.path, k, params[k], v)
					}
				}
			}
		})
	}
}
