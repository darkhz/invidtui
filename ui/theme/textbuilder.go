package theme

import (
	"fmt"
	"strings"
)

// ThemeTextBuilder describes a text builder for themed text.
type ThemeTextBuilder struct {
	builder strings.Builder
	context ThemeContext
}

// NewTextBuilder returns a new theme text builder.
func NewTextBuilder(context ThemeContext) ThemeTextBuilder {
	return ThemeTextBuilder{
		builder: strings.Builder{},
		context: context,
	}
}

// Start appends the region and starting style tag to the text builder.
func (t *ThemeTextBuilder) Start(item ThemeItem, region string) {
	_, tag, ok := GetThemeSetting(ThemeProperty{
		Context: t.context,
		Item:    item,
	})
	if !ok {
		return
	}

	fmt.Fprintf(t, "[\"%s,%s;%s\"]%s", t.context, item, region, tag)
}

// Append applies the start tags, appends the text to the text builder and applies the end tags.
func (t *ThemeTextBuilder) Append(item ThemeItem, region, text string) {
	t.Start(item, region)
	t.builder.WriteString(text)
	t.Finish()
}

// AppendText appends text to the text builder.
func (t *ThemeTextBuilder) AppendText(text string) {
	t.builder.WriteString(text)
}

// Format applies the start tags, appends formatted text to the text builder and applies the end tags.
func (t *ThemeTextBuilder) Format(item ThemeItem, region, format string, values ...any) {
	t.Start(item, region)
	fmt.Fprintf(t, format, values...)
	t.Finish()
}

// Get returns the text from the text builder.
func (t *ThemeTextBuilder) Get(noclear ...struct{}) string {
	if noclear == nil {
		defer t.builder.Reset()
	}

	return t.builder.String()
}

// Finish applies the end tags to the text builder.
func (t *ThemeTextBuilder) Finish() {
	fmt.Fprintf(&t.builder, "[-:-:-][\"\"]")
}

// Len returns the size of the text builder.
func (t *ThemeTextBuilder) Len() int {
	return t.builder.Len()
}

// Write writes to the text builder.
func (t *ThemeTextBuilder) Write(b []byte) (int, error) {
	return t.builder.Write(b)
}
