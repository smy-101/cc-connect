package feishu

import (
	"github.com/smy-101/cc-connect/internal/core"
)

// renderCardMap converts a Card to Feishu card JSON as a map[string]any.
// This follows the Feishu card v1 format.
func renderCardMap(card *core.Card) map[string]any {
	if card == nil {
		return nil
	}

	result := map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
		},
	}

	// Render header
	if card.Header != nil {
		result["header"] = map[string]any{
			"title": map[string]any{
				"content": card.Header.Title,
				"tag":     "plain_text",
			},
			"template": card.Header.Template,
		}
	}

	// Render elements
	var elements []map[string]any
	for _, elem := range card.Elements {
		switch e := elem.(type) {
		case core.CardMarkdown:
			elements = append(elements, map[string]any{
				"tag": "div",
				"text": map[string]any{
					"content": e.Content,
					"tag":     "lark_md",
				},
			})

		case core.CardActions:
			var actions []map[string]any
			for _, btn := range e.Buttons {
				actions = append(actions, renderButton(btn))
			}
			elements = append(elements, map[string]any{
				"tag":     "action",
				"actions": actions,
			})

		case core.CardDivider:
			elements = append(elements, map[string]any{
				"tag": "hr",
			})

		case core.CardNote:
			elements = append(elements, map[string]any{
				"tag": "note",
				"elements": []map[string]any{
					{
						"tag": "plain_text",
						"text": map[string]any{
							"content": e.Content,
						},
					},
				},
			})
		}
	}
	result["elements"] = elements

	return result
}

// renderButton converts a CardButton to Feishu button format.
func renderButton(btn core.CardButton) map[string]any {
	// Map button types to Feishu button styles
	style := "default"
	switch btn.Type {
	case "primary":
		style = "primary"
	case "danger":
		style = "danger"
	}

	return map[string]any{
		"tag": "button",
		"text": map[string]any{
			"content": btn.Text,
			"tag":     "plain_text",
		},
		"type":  style,
		"value": map[string]any{"action": btn.Value},
	}
}
