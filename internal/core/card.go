// Card represents an interactive card message
package core

import (
	"fmt"
	"strings"
)

// CardElement is the interface for card elements
type CardElement interface {
	ElementType() string
}

// CardHeader represents a card header
type CardHeader struct {
	Title    string `json:"title"`
	Template string `json:"template,omitempty"` // "blue", "wathet", "red", "green"
	Color    string `json:"color,omitempty"`
}

// CardMarkdown represents a markdown content
type CardMarkdown struct {
	Content string `json:"content"`
}

// ElementType implements CardElement
func (m CardMarkdown) ElementType() string {
	return "markdown"
}

// CardButton represents a clickable button
type CardButton struct {
	Text  string `json:"text"`
	Type  string `json:"type"`  // "primary", "default", "danger"
	Value string `json:"value"`
}

// PrimaryBtn creates a primary button
func PrimaryBtn(text, value string) CardButton {
	return CardButton{Text: text, Type: "primary", Value: value}
}

// DefaultBtn creates a default button
func DefaultBtn(text, value string) CardButton {
	return CardButton{Text: text, Type: "default", Value: value}
}

// DangerBtn creates a danger button
func DangerBtn(text, value string) CardButton {
	return CardButton{Text: text, Type: "danger", Value: value}
}

// CardActions represents a row of buttons
type CardActions struct {
	Buttons []CardButton `json:"buttons"`
}

// ElementType implements CardElement
func (a CardActions) ElementType() string {
	return "action"
}

// CardDivider represents a horizontal divider
type CardDivider struct{}

// ElementType implements CardElement
func (d CardDivider) ElementType() string {
	return "divider"
}

// CardNote represents a note/footer text
type CardNote struct {
	Content string `json:"content"`
}

// ElementType implements CardElement
func (n CardNote) ElementType() string {
	return "note"
}

// Card represents a complete interactive card
type Card struct {
	Header   *CardHeader   `json:"header,omitempty"`
	Elements []CardElement `json:"elements"`
}

// CardBuilder provides a fluent interface for building cards
type CardBuilder struct {
	card *Card
}

// NewCard creates a new CardBuilder
func NewCard() *CardBuilder {
	return &CardBuilder{
		card: &Card{},
	}
}

// Title sets the card title
func (b *CardBuilder) Title(title, template string) *CardBuilder {
	b.card.Header = &CardHeader{Title: title}
	if template != "" {
		b.card.Header.Template = template
	}
	return b
}

// Color sets the header color
func (b *CardBuilder) Color(color string) *CardBuilder {
	if b.card.Header != nil {
		b.card.Header.Color = color
	}
	return b
}

// Markdown adds a markdown element
func (b *CardBuilder) Markdown(content string) *CardBuilder {
	b.card.Elements = append(b.card.Elements, CardMarkdown{Content: content})
	return b
}

// Buttons adds a row of buttons
func (b *CardBuilder) Buttons(buttons ...CardButton) *CardBuilder {
	b.card.Elements = append(b.card.Elements, CardActions{Buttons: buttons})
	return b
}

// ButtonsEqual adds a row of buttons with equal width
func (b *CardBuilder) ButtonsEqual(buttons ...CardButton) *CardBuilder {
	return b.Buttons(buttons...)
}

// Divider adds a horizontal divider
func (b *CardBuilder) Divider() *CardBuilder {
	b.card.Elements = append(b.card.Elements, CardDivider{})
	return b
}

// Note adds a note/footer text
func (b *CardBuilder) Note(content string) *CardBuilder {
	b.card.Elements = append(b.card.Elements, CardNote{Content: content})
	return b
}

// Build returns the built card
func (b *CardBuilder) Build() *Card {
	return b.card
}

// RenderText renders the card as a fallback text representation
func (c *Card) RenderText() string {
	var sb strings.Builder

	if c.Header != nil && c.Header.Title != "" {
		sb.WriteString("🤖 " + c.Header.Title + "\n")
	} else {
		sb.WriteString("🤖 （无标题）\n")
	}
	sb.WriteString("\n")

	for _, elem := range c.Elements {
		switch e := elem.(type) {
		case CardMarkdown:
			sb.WriteString(e.Content + "\n")
		case CardActions:
			for _, btn := range e.Buttons {
				sb.WriteString(fmt.Sprintf("[%s] ", btn.Text))
			}
			sb.WriteString("\n")
		case CardDivider:
			sb.WriteString("---\n")
		case CardNote:
			sb.WriteString(e.Content + "\n")
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// CardSender interface for sending cards
type CardSender interface {
	SendCard(ctx interface{}, replyCtx interface{}, card *Card) error
	ReplyCard(ctx interface{}, replyCtx interface{}, card *Card) error
}
