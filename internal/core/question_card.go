// Question card generation for AskUserQuestion
package core

// QuestionCardInput represents input for building a question card
type QuestionCardInput struct {
	Question    string
	Header      string
	RequestID   string
	Options     []QuestionOption
	MultiSelect bool
}

// QuestionOption represents an option in a question
type QuestionOption struct {
	Label       string
	Description string
}

// BuildQuestionCard builds a card for AskUserQuestion
func BuildQuestionCard(q QuestionCardInput) *Card {
	header := q.Header
	if header == "" {
		header = "Question"
	}

	builder := NewCard().Title(header, "blue")

	// Build question content
	content := q.Question
	if q.MultiSelect {
		content = q.Question + "\n\n*（多选：可选择多个选项）*"
	} else if len(q.Options) == 0 {
		content = q.Question + "\n\n*（请直接回复文本答案）*"
	}
	builder.Markdown(content)

	// Add buttons for options
	if len(q.Options) > 0 {
		var buttons []CardButton
		for _, opt := range q.Options {
			value := "ans:" + q.RequestID + ":" + opt.Label
			buttons = append(buttons, DefaultBtn(opt.Label, value))
		}
		builder.Buttons(buttons...)
	}

	return builder.Build()
}
