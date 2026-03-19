package claudecode

import (
	"reflect"
	"testing"
)

// TestParseUserQuestions tests parsing questions from AskUserQuestion tool input
func TestParseUserQuestions(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected []UserQuestion
	}{
		{
			name:  "empty input",
			input: map[string]interface{}{},
			expected: nil,
		},
		{
			name: "nil questions",
			input: map[string]interface{}{
				"questions": nil,
			},
			expected: nil,
		},
		{
			name: "single question without options",
			input: map[string]interface{}{
				"questions": []interface{}{
					map[string]interface{}{
						"question": "What is your name?",
					},
				},
			},
			expected: []UserQuestion{
				{Question: "What is your name?"},
			},
		},
		{
			name: "single question with header",
			input: map[string]interface{}{
				"questions": []interface{}{
					map[string]interface{}{
						"question": "Which database?",
						"header":   "Database",
					},
				},
			},
			expected: []UserQuestion{
				{Question: "Which database?", Header: "Database"},
			},
		},
		{
			name: "single question with options",
			input: map[string]interface{}{
				"questions": []interface{}{
					map[string]interface{}{
						"question": "Which database?",
						"header":   "Database",
						"options": []interface{}{
							map[string]interface{}{
								"label":      "PostgreSQL",
								"description": "PostgreSQL database",
							},
							map[string]interface{}{
								"label":      "MySQL",
								"description": "MySQL database",
							},
						},
					},
				},
			},
			expected: []UserQuestion{
				{
					Question: "Which database?",
					Header:   "Database",
					Options: []UserQuestionOption{
						{Label: "PostgreSQL", Description: "PostgreSQL database"},
						{Label: "MySQL", Description: "MySQL database"},
					},
				},
			},
		},
		{
			name: "multiSelect question",
			input: map[string]interface{}{
				"questions": []interface{}{
					map[string]interface{}{
						"question":   "Select features:",
						"multiSelect": true,
						"options": []interface{}{
							map[string]interface{}{
								"label": "Auth",
							},
							map[string]interface{}{
								"label": "Logging",
							},
						},
					},
				},
			},
			expected: []UserQuestion{
				{
					Question:   "Select features:",
					MultiSelect: true,
					Options: []UserQuestionOption{
						{Label: "Auth"},
						{Label: "Logging"},
					},
				},
			},
		},
		{
			name: "multiple questions",
			input: map[string]interface{}{
				"questions": []interface{}{
					map[string]interface{}{
						"question": "Question 1?",
						"header":   "H1",
					},
					map[string]interface{}{
						"question": "Question 2?",
						"header":   "H2",
					},
				},
			},
			expected: []UserQuestion{
				{Question: "Question 1?", Header: "H1"},
				{Question: "Question 2?", Header: "H2"},
			},
		},
		{
			name: "skips invalid question entries",
			input: map[string]interface{}{
				"questions": []interface{}{
					"invalid string",
					map[string]interface{}{
						"question": "Valid question?",
					},
					123,
				},
			},
			expected: []UserQuestion{
				{Question: "Valid question?"},
			},
		},
		{
			name: "skips invalid option entries",
			input: map[string]interface{}{
				"questions": []interface{}{
					map[string]interface{}{
						"question": "Question?",
						"options": []interface{}{
							"invalid string",
							map[string]interface{}{
								"label": "Valid option",
							},
						},
					},
				},
			},
			expected: []UserQuestion{
				{
					Question: "Question?",
					Options: []UserQuestionOption{
						{Label: "Valid option"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUserQuestions(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseUserQuestions() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}
