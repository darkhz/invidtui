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

// ThemePrimitiveConfig stores the configuration for all primitives.
type ThemePrimitiveConfig struct {
	// Version is the global theme updater version.
	// All ThemeProperties check this version before updating its attached primitive.
	Version int

	// Draw draws the primitive onto the screen.
	Draw func(p tview.Primitive)
}

var pconfig ThemePrimitiveConfig

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
	var itemCount int
	var buttonCount int

	form := tview.NewForm()
	form.SetItemAttributesFunc(func(item tview.FormItem, labelWidth int) {
		if itemCount == form.GetFormItemCount() {
			return
		}

		itemCount++
		applyTheme(item, property, labelWidth)
	})
	form.SetButtonAttributesFunc(func(button *tview.Button) {
		if buttonCount == form.GetButtonCount() {
			return
		}

		buttonCount++
		applyTheme(button, property)
	})
	form.SetBlurFunc(func() {
		itemCount = 0
		buttonCount = 0
	})
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
	inputfield.SetLabel(GetLabel(
		property.SetItem(ThemeInputLabel), label, true),
	)
	WrapDrawFunc(inputfield, property, func(_ tcell.Screen, _, _, _, _ int) (int, int, int, int) {
		return inputfield.GetInnerRect()
	})

	return inputfield
}

// NewDropDown returns a new dropdown primitive.
func NewDropDown(property ThemeProperty, label string) *tview.DropDown {
	dropdown := tview.NewDropDown()
	dropdown.SetLabel(GetLabel(
		property.SetItem(ThemeListLabel), label, true),
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

// SetDrawFunc sets the primitive's draw handler.
func SetDrawFunc(draw func(p tview.Primitive)) {
	pconfig.Draw = draw
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

	property.Version = pconfig.Version

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
func applyTheme(primitive tview.Primitive, t ThemeProperty, labelWidth ...int) {
	defer func() {
		if pconfig.Draw != nil && labelWidth == nil {
			pconfig.Draw(primitive)
		}
	}()

	bgProperty := ThemeProperty{
		Context: t.Context,
		Item:    ThemeBackground,
	}
	borderProperty := ThemeProperty{
		Context: t.Context,
		Item:    ThemeBorder,
	}
	if t.Item == ThemePopupBackground {
		bgProperty.Item = ThemePopupBackground
		borderProperty.Item = ThemePopupBorder
	}

	bgStyle, _, _ := GetThemeSetting(bgProperty)
	borderStyle, _, _ := GetThemeSetting(borderProperty)
	_, bgColor, _ := bgStyle.Decompose()

	if p, ok := primitive.(ThemePrimitive); ok {
		p.SetBorderStyle(borderStyle)
		p.SetBackgroundColor(bgColor)

		if title := p.GetTitle(); title != "" {
			p.SetTitle(GetThemedRegions(title))
		}
	}

	if labelWidth != nil {
		if p, ok := primitive.(tview.FormItem); ok {
			p.SetFormAttributes(labelWidth[0], 0, bgColor, 0, 0)
		}
	}

	propMap := map[string]ThemeItem{
		"label":   ThemeFormLabel,
		"field":   ThemeFormField,
		"options": ThemeFormOptions,
	}

	switch p := primitive.(type) {
	case *tview.TextView:
		p.SetText(GetThemedRegions(p.GetText(false)))

	case *tview.Button:
		style, _, ok := GetThemeSetting(ThemeProperty{
			Context: t.Context,
			Item:    ThemeFormButton,
		})
		if ok {
			p.SetStyle(style)
		}

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

	case *tview.Checkbox:
		for _, item := range []ThemeItem{
			ThemeFormLabel,
			ThemeFormField,
		} {
			property := ThemeProperty{
				Context: t.Context,
				Item:    item,
			}

			style, _, ok := GetThemeSetting(property)
			if ok {
				if _, bg, _ := style.Decompose(); bg == 0 {
					style = style.Background(bgColor)
				}
			}

			switch item {
			case ThemeFormLabel:
				p.SetLabel(GetThemedLabel(
					property, p.GetLabel(), false,
				))

			case ThemeFormField:
				fg, bg, _ := style.Decompose()

				p.SetFieldTextColor(fg)
				p.SetFieldBackgroundColor(bg)
			}
		}

	case *tview.InputField:
		if labelWidth == nil {
			propMap["label"] = ThemeInputLabel
			propMap["field"] = ThemeInputField
		}

		for name, item := range propMap {
			property := ThemeProperty{
				Context: t.Context,
				Item:    item,
			}

			style, _, ok := GetThemeSetting(property)
			if !ok {
				continue
			}

			_, bg, _ := style.Decompose()
			if bg == 0 {
				style = style.Background(bgColor)
			}

			switch name {
			case "label":
				p.SetLabelStyle(tcell.Style{}.Background(bg))
				if p.GetLabel() != "" {
					p.SetLabel(GetThemedLabel(
						property, p.GetLabel(), labelWidth == nil,
					))
				}

			case "field":
				p.SetFieldStyle(style)
			}
		}

	case *tview.DropDown:
		if labelWidth == nil {
			propMap["label"] = ThemeListLabel
			propMap["field"] = ThemeListField
			propMap["options"] = ThemeListOptions
		}

		for name, item := range propMap {
			property := ThemeProperty{
				Context: t.Context,
				Item:    item,
			}

			style, _, ok := GetThemeSetting(property)
			if !ok {
				continue
			}

			fg, bg, _ := style.Decompose()

			switch name {
			case "label":
				p.SetLabel(GetThemedLabel(
					property, p.GetLabel(), false,
				))

			case "field":
				p.SetFieldTextColor(fg)
				p.SetFieldBackgroundColor(bg)

			case "options":
				p.List().SetMainTextColor(fg)
				p.List().SetBackgroundColor(bg)
			}
		}
	}
}
