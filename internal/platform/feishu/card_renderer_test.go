package feishu

import (
	"testing"

	"github.com/smy-101/cc-connect/internal/core"
)

func TestRenderCardMap(t *testing.T) {
	tests := []struct {
		name     string
		card     *core.Card
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "card with header only",
			card: core.NewCard().
				Title("Test Title", "blue").
				Build(),
			validate: func(t *testing.T, result map[string]any) {
				config, ok := result["config"].(map[string]any)
				if !ok {
					t.Fatal("config not found")
				}
				if config["wide_screen_mode"] != true {
					t.Error("wide_screen_mode should be true")
				}

				header, ok := result["header"].(map[string]any)
				if !ok {
					t.Fatal("header not found")
				}
				title, ok := header["title"].(map[string]any)
				if !ok {
					t.Fatal("header title not found")
				}
				if title["content"] != "Test Title" {
					t.Errorf("expected title 'Test Title', got %v", title["content"])
				}
				if header["template"] != "blue" {
					t.Errorf("expected template 'blue', got %v", header["template"])
				}
			},
		},
		{
			name: "card with markdown element",
			card: core.NewCard().
				Title("Test", "blue").
				Markdown("**工具**: Bash\n**命令**: `npm install`").
				Build(),
			validate: func(t *testing.T, result map[string]any) {
				elements, ok := result["elements"].([]map[string]any)
				if !ok {
					t.Fatal("elements not found or wrong type")
				}
				if len(elements) != 1 {
					t.Fatalf("expected 1 element, got %d", len(elements))
				}
				if elements[0]["tag"] != "div" {
					t.Errorf("expected tag 'div', got %v", elements[0]["tag"])
				}
				text, ok := elements[0]["text"].(map[string]any)
				if !ok {
					t.Fatal("text not found")
				}
				if text["content"] != "**工具**: Bash\n**命令**: `npm install`" {
					t.Errorf("unexpected markdown content: %v", text["content"])
				}
				if text["tag"] != "lark_md" {
					t.Errorf("expected text tag 'lark_md', got %v", text["tag"])
				}
			},
		},
		{
			name: "card with action buttons",
			card: core.NewCard().
				Title("Test", "blue").
				Buttons(
					core.PrimaryBtn("✅ 允许", "perm:allow:req123"),
					core.DangerBtn("❌ 拒绝", "perm:deny:req123"),
				).
				Build(),
			validate: func(t *testing.T, result map[string]any) {
				elements, ok := result["elements"].([]map[string]any)
				if !ok {
					t.Fatal("elements not found or wrong type")
				}
				if len(elements) != 1 {
					t.Fatalf("expected 1 element, got %d", len(elements))
				}
				if elements[0]["tag"] != "action" {
					t.Errorf("expected tag 'action', got %v", elements[0]["tag"])
				}
				actions, ok := elements[0]["actions"].([]map[string]any)
				if !ok {
					t.Fatal("actions not found")
				}
				if len(actions) != 2 {
					t.Fatalf("expected 2 buttons, got %d", len(actions))
				}

				// Check first button (primary)
				btn1 := actions[0]
				if btn1["tag"] != "button" {
					t.Errorf("expected button tag 'button', got %v", btn1["tag"])
				}
				btn1Text := btn1["text"].(map[string]any)
				if btn1Text["content"] != "✅ 允许" {
					t.Errorf("expected button text '✅ 允许', got %v", btn1Text["content"])
				}

				btn1Value := btn1["value"].(map[string]any)
				if btn1Value["action"] != "perm:allow:req123" {
					t.Errorf("expected action 'perm:allow:req123', got %v", btn1Value["action"])
				}
			},
		},
		{
			name: "card with divider",
			card: core.NewCard().
				Title("Test", "blue").
				Markdown("Content").
				Divider().
				Build(),
			validate: func(t *testing.T, result map[string]any) {
				elements := result["elements"].([]map[string]any)
				if len(elements) != 2 {
					t.Fatalf("expected 2 elements, got %d", len(elements))
				}
				if elements[1]["tag"] != "hr" {
					t.Errorf("expected tag 'hr' for divider, got %v", elements[1]["tag"])
				}
			},
		},
		{
			name: "card with note",
			card: core.NewCard().
				Title("Test", "blue").
				Markdown("Content").
				Note("回复 A 允许，D 拒绝").
				Build(),
			validate: func(t *testing.T, result map[string]any) {
				elements := result["elements"].([]map[string]any)
				if len(elements) != 2 {
					t.Fatalf("expected 2 elements, got %d", len(elements))
				}
				if elements[1]["tag"] != "note" {
					t.Errorf("expected tag 'note', got %v", elements[1]["tag"])
				}
				elements1 := elements[1]["elements"].([]map[string]any)
				if len(elements1) != 1 {
					t.Fatalf("expected 1 element in note, got %d", len(elements1))
				}
				text := elements1[0]["text"].(map[string]any)
				if text["content"] != "回复 A 允许，D 拒绝" {
					t.Errorf("unexpected note content: %v", text["content"])
				}
			},
		},
		{
			name: "permission request card",
			card: core.NewCard().
				Title("🤖 Claude 需要您的确认", "blue").
				Markdown("**工具**: Bash\n**命令**: `npm install`").
				ButtonsEqual(
					core.PrimaryBtn("✅ 允许", "perm:allow:req123"),
					core.DangerBtn("❌ 拒绝", "perm:deny:req123"),
				).
				Note("回复 A 允许，D 拒绝").
				Build(),
			validate: func(t *testing.T, result map[string]any) {
				// Verify config
				config := result["config"].(map[string]any)
				if config["wide_screen_mode"] != true {
					t.Error("wide_screen_mode should be true")
				}

				// Verify header
				header := result["header"].(map[string]any)
				title := header["title"].(map[string]any)
				if title["content"] != "🤖 Claude 需要您的确认" {
					t.Errorf("unexpected title: %v", title["content"])
				}

				// Verify elements
				elements := result["elements"].([]map[string]any)
				if len(elements) != 3 {
					t.Fatalf("expected 3 elements (markdown, action, note), got %d", len(elements))
				}

				// Verify markdown
				if elements[0]["tag"] != "div" {
					t.Errorf("expected first element to be div, got %v", elements[0]["tag"])
				}

				// Verify action buttons
				if elements[1]["tag"] != "action" {
					t.Errorf("expected second element to be action, got %v", elements[1]["tag"])
				}
				actions := elements[1]["actions"].([]map[string]any)
				if len(actions) != 2 {
					t.Errorf("expected 2 buttons, got %d", len(actions))
				}

				// Verify note
				if elements[2]["tag"] != "note" {
					t.Errorf("expected third element to be note, got %v", elements[2]["tag"])
				}
			},
		},
		{
			name: "ask user question card",
			card: core.NewCard().
				Title("🤖 Claude 问您", "blue").
				Markdown("您希望使用哪种数据库？").
				ButtonsEqual(
					core.DefaultBtn("PostgreSQL", "ans:req123:PostgreSQL"),
					core.DefaultBtn("MySQL", "ans:req123:MySQL"),
					core.DefaultBtn("SQLite", "ans:req123:SQLite"),
				).
				Build(),
			validate: func(t *testing.T, result map[string]any) {
				elements := result["elements"].([]map[string]any)
				if len(elements) != 2 {
					t.Fatalf("expected 2 elements, got %d", len(elements))
				}

				// Verify action buttons (3 options)
				if elements[1]["tag"] != "action" {
					t.Errorf("expected second element to be action, got %v", elements[1]["tag"])
				}
				actions := elements[1]["actions"].([]map[string]any)
				if len(actions) != 3 {
					t.Errorf("expected 3 buttons, got %d", len(actions))
				}

				// Verify first button value
				btn1Value := actions[0]["value"].(map[string]any)
				if btn1Value["action"] != "ans:req123:PostgreSQL" {
					t.Errorf("expected action 'ans:req123:PostgreSQL', got %v", btn1Value["action"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderCardMap(tt.card)
			if result == nil {
				t.Fatal("renderCardMap returned nil")
			}
			tt.validate(t, result)
		})
	}
}
