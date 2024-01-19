package theme

import (
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// ThemePrimitive describes a theme primitive.
type ThemePrimitive interface {
	SetDrawFunc(func(s tcell.Screen, x, y, width, height int) (int, int, int, int)) *tview.Box

	SetBorderStyle(tcell.Style) *tview.Box
	SetBackgroundColor(tcell.Color) *tview.Box

	GetTitle() string
	SetTitle(title string) *tview.Box
}

// ThemeVersion is the global theme updater version.
// All ThemeProperties check this version before updating its attached primitive.
var ThemeVersion int

// NewBox returns a new box primitive.
func NewBox(property ThemeProperty) *tview.Box {
	box := tview.NewBox()
	WrapDrawFunc(box, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return box.GetInnerRect()
	})

	return box
}

// NewFlex returns a new flex primitive.
func NewFlex(property ThemeProperty) *tview.Flex {
	flex := tview.NewFlex()
	WrapDrawFunc(flex, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return flex.GetInnerRect()
	})

	return flex
}

// NewForm returns a new form primitive.
func NewForm(property ThemeProperty) *tview.Form {
	form := tview.NewForm()
	WrapDrawFunc(form, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return form.GetInnerRect()
	})

	return form
}

// NewTable returns a new table primitive.
func NewTable(property ThemeProperty) *tview.Table {
	table := tview.NewTable()
	WrapDrawFunc(table, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return table.GetInnerRect()
	})

	return table
}

// NewImage returns a new image primitive.
func NewImage(property ThemeProperty) *tview.Image {
	image := tview.NewImage()
	WrapDrawFunc(image, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return image.GetInnerRect()
	})

	return image
}

// NewTreeView returns a new treeview primitive.
func NewTreeView(property ThemeProperty) *tview.TreeView {
	treeview := tview.NewTreeView()

	WrapDrawFunc(treeview, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return treeview.GetInnerRect()
	})

	return treeview
}

// NewInputField returns a new inputfield primitive.
func NewInputField(property ThemeProperty, label string) *tview.InputField {
	inputfield := tview.NewInputField()
	inputfield.SetLabel(label)
	inputfield.SetLabelWidth(
		tview.TaggedStringWidth(inputfield.GetLabel()) + 1,
	)
	WrapDrawFunc(inputfield, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return inputfield.GetInnerRect()
	})

	return inputfield
}

// NewDropDown returns a new dropdown primitive.
func NewDropDown(property ThemeProperty, label string) *tview.DropDown {
	dropdown := tview.NewDropDown()
	dropdown.SetLabel(label)
	dropdown.SetLabelWidth(
		tview.TaggedStringWidth(dropdown.GetLabel()) + 1,
	)
	WrapDrawFunc(dropdown, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return dropdown.GetInnerRect()
	})

	return dropdown
}

// NewTextView returns a new textview primitive.
func NewTextView(property ThemeProperty) *tview.TextView {
	textview := tview.NewTextView()
	textview.SetWrap(true)
	textview.SetRegions(true)
	textview.SetDynamicColors(true)
	WrapDrawFunc(textview, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return textview.GetInnerRect()
	})

	return textview
}

// NewPages returns a new pages primitive.
func NewPages(property ThemeProperty) *tview.Pages {
	pages := tview.NewPages()
	WrapDrawFunc(pages, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return pages.GetInnerRect()
	})

	return pages
}

// NewTableCell returns a new tablecell.
func NewTableCell(context ThemeContext, item ThemeItem, text string) *tview.TableCell {
	return tview.NewTableCell(SetTextStyle("region", text, context, item))
}

// WrapDrawFunc wraps the primitive's DrawFunc with SetThemeProperty.
func WrapDrawFunc(
	primitive tview.Primitive, property ThemeProperty,
	drawFunc func(s tcell.Screen, x, y, width, height int) (int, int, int, int),
	noapply ...struct{},
) {
	p, ok := primitive.(ThemePrimitive)
	if !ok {
		return
	}

	property.Version = ThemeVersion

	p.SetDrawFunc(func(s tcell.Screen, x, y, width, height int) (int, int, int, int) {
		SetThemeProperty(primitive, &property)

		return drawFunc(s, x, y, width, height)
	})

	if noapply == nil {
		applyTheme(primitive, property)
	}
}

// applyTheme applies the theme to the primitive.
//
//gocyclo:ignore
func applyTheme(primitive tview.Primitive, t ThemeProperty) {
	bgProperty := ThemeProperty{
		Context: t.Context,
		Item:    ThemeBackground,
	}
	if t.Item == ThemePopupBackground {
		bgProperty.Item = t.Item
	}

	bgStyle, _, _ := GetThemeSetting(bgProperty)
	borderStyle, _, _ := GetThemeSetting(ThemeProperty{
		Context: t.Context,
		Item:    ThemeBorder,
	})

	_, bgColor, _ := bgStyle.Decompose()

	if p, ok := primitive.(ThemePrimitive); ok {
		p.SetBorderStyle(borderStyle)
		p.SetBackgroundColor(bgColor)

		if title := p.GetTitle(); title != "" {
			p.SetTitle(GetThemedRegions(title))
		}
	}

	switch p := primitive.(type) {
	case *tview.TextView:
		p.SetText(GetThemedRegions(p.GetText(false)))

	case *tview.Table:
		style, _, ok := GetThemeSetting(ThemeProperty{
			Context: t.Context,
			Item:    ThemeSelector,
		})
		if ok {
			p.SetSelectedStyle(style)
		}

		rows, cols := p.GetRowCount(), p.GetColumnCount()
		for col := 0; col < cols; col++ {
			for row := 0; row < rows; row++ {
				cell := p.GetCell(row, col)

				cell.SetText(GetThemedRegions(cell.Text))
			}
		}

	case *tview.TreeView:
		style, _, ok := GetThemeSetting(ThemeProperty{
			Context: t.Context,
			Item:    ThemeSelector,
		})
		if ok {
			p.SetSelectedStyle(style)
		}

		if root := p.GetRoot(); root != nil {
			root.Walk(func(node, parent *tview.TreeNode) bool {
				if node != nil {
					node.SetText(GetThemedRegions(node.GetText()))

				}
				return true
			})
		}

	case *tview.InputField:
		for item, styleFunc := range map[ThemeItem]func(s tcell.Style) *tview.InputField{
			ThemeInputLabel: p.SetLabelStyle,
			ThemeInputField: p.SetFieldStyle,
		} {
			style, _, ok := GetThemeSetting(ThemeProperty{
				Context: t.Context,
				Item:    item,
			})
			if ok {
				if _, bg, _ := style.Decompose(); bg == 0 {
					style = style.Background(bgColor)
				}

				styleFunc(style)
			}
		}

	case *tview.DropDown:
		for _, item := range []ThemeItem{
			ThemeListLabel,
			ThemeListField,
			ThemeListOptions,
		} {
			style, _, ok := GetThemeSetting(ThemeProperty{
				Context: t.Context,
				Item:    item,
			})
			if !ok {
				continue
			}

			fg, bg, _ := style.Decompose()

			switch item {
			case ThemeListLabel:
				p.SetLabelColor(fg)

			case ThemeListField:
				p.SetFieldTextColor(fg)
				p.SetFieldBackgroundColor(bg)

			case ThemeListOptions:
				p.List().SetMainTextColor(fg)
				p.List().SetBackgroundColor(bg)
			}
		}

	case *tview.Form:
		for _, item := range []ThemeItem{
			ThemeInputLabel,
			ThemeInputField,
			ThemeListLabel,
			ThemeListField,
			ThemeListOptions,
			ThemeButton,
		} {
			style, _, ok := GetThemeSetting(ThemeProperty{
				Context: t.Context,
				Item:    item,
			})
			if !ok {
				continue
			}

			fg, bg, _ := style.Decompose()

			switch item {
			case ThemeListField, ThemeInputField:
				p.SetFieldBackgroundColor(bg)
				p.SetFieldTextColor(fg)

			case ThemeInputLabel, ThemeListLabel:
				p.SetLabelColor(fg)

			case ThemeButton:
				p.SetButtonStyle(style)
			}
		}
	}
}
