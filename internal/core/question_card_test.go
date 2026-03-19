package core

import (
	"testing"
)

// TestBuildQuestionCard tests building cards for AskUserQuestion
func TestBuildQuestionCard(t *testing.T) {
	t.Run("single select question", func(t *testing.T) {
		question := QuestionCardInput{
			Question: "Which database?",
			Header:   "Database",
			RequestID: "req-123",
			Options: []QuestionOption{
				{Label: "PostgreSQL", Description: "PostgreSQL database"},
				{Label: "MySQL", Description: "MySQL database"},
				{Label: "SQLite", Description: "SQLite database"},
			},
			MultiSelect: false,
		}

		card := BuildQuestionCard(question)

		if card.Header == nil {
			t.Fatal("Expected card header")
		}
		if card.Header.Title != "Database" {
			t.Errorf("Expected title 'Database', got %q", card.Header.Title)
		}

		// Should have markdown + actions
		if len(card.Elements) < 2 {
			t.Errorf("Expected at least 2 elements, got %d", len(card.Elements))
		}

		// Check markdown contains question
		md, ok := card.Elements[0].(CardMarkdown)
		if !ok {
			t.Fatal("Expected first element to be markdown")
		}
		if md.Content != "Which database?" {
			t.Errorf("Expected content 'Which database?', got %q", md.Content)
		}

		// Check actions contain options as buttons
		actions, ok := card.Elements[1].(CardActions)
		if !ok {
			t.Fatal("Expected second element to be actions")
		}
		if len(actions.Buttons) != 3 {
			t.Errorf("Expected 3 buttons, got %d", len(actions.Buttons))
		}

		// Check button values format: "ans:req-123:PostgreSQL"
		expectedValue := "ans:req-123:PostgreSQL"
		if actions.Buttons[0].Value != expectedValue {
			t.Errorf("Expected button value %q, got %q", expectedValue, actions.Buttons[0].Value)
		}
	})

	t.Run("multi select question", func(t *testing.T) {
		question := QuestionCardInput{
			Question: "Select features:",
			Header:   "Features",
			RequestID: "req-456",
			Options: []QuestionOption{
				{Label: "Auth"},
				{Label: "Logging"},
				{Label: "Metrics"},
			},
			MultiSelect: true,
		}

		card := BuildQuestionCard(question)

		// Should have markdown with multi-select hint
		md, ok := card.Elements[0].(CardMarkdown)
		if !ok {
			t.Fatal("Expected first element to be markdown")
		}
		// Multi-select should have a hint
		if md.Content == "Select features:" {
			t.Error("Expected multi-select hint in content")
		}
	})

	t.Run("open question without options", func(t *testing.T) {
		question := QuestionCardInput{
			Question:    "What is your name?",
			Header:      "Name",
			RequestID:   "req-789",
			MultiSelect: false,
		}

		card := BuildQuestionCard(question)

		// Should have markdown with text input hint
		md, ok := card.Elements[0].(CardMarkdown)
		if !ok {
			t.Fatal("Expected first element to be markdown")
		}
		// Should have text input hint
		if md.Content == "What is your name?" {
			t.Error("Expected text input hint for open question")
		}
	})

	t.Run("question without header uses default", func(t *testing.T) {
		question := QuestionCardInput{
			Question:    "What do you want?",
			RequestID:   "req-000",
			MultiSelect: false,
		}

		card := BuildQuestionCard(question)

		if card.Header.Title != "Question" {
			t.Errorf("Expected default title 'Question', got %q", card.Header.Title)
		}
	})
}
