//go:build !wasm

package panel

import (
	"webtyp.com/css"
	"webtyp.com/widget"
	"webtyp.com/widget/style"
)

func (p *Panel) WidgetName() widget.Name { return widget.Name("iam-panel") }
func (p *Panel) WidgetKind() widget.Kind { return widget.Region }

// RenderCSS declares layout styles for the panel.
func (p *Panel) RenderCSS() *css.Stylesheet {
	return style.For(p).
		Root(
			style.Stack(style.Space4),
			style.Pad(style.Space4),
		).
		Stylesheet()
}
