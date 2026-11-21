//go:build !wasm

package ui

import (
	"webtyp.com/css"
	"webtyp.com/widget/style"
)

// RenderCSS styles the multi-role assignments screen using the widget style DSL.
func (v *AssignmentsView) RenderCSS() *css.Stylesheet {
	return style.For(v).
		Root(
			style.Stack(style.Space4),
			style.Pad(style.Space4),
			style.KeepSize(),
		).
		Part(PartAssignHeader,
			style.Stack(style.Space3),
		).
		Part(PartAssignInvite,
			style.Row(style.Space2),
			style.CenterContent(),
		).
		Part(PartAssignList,
			style.Stack(style.Space3),
		).
		Part(PartAssignCard,
			style.Stack(style.Space2),
			style.As(style.Page),
			style.Pad(style.Space3),
			style.Round(style.RadiusMd),
		).
		Part(PartAssignPerson,
			style.Row(style.Space2),
			style.CenterContent(),
		).
		Part(PartAssignEmail,
			style.FontSize(style.TextSm),
		).
		Part(PartAssignChips,
			style.Row(style.Space2),
		).
		Part(PartAssignChip,
			style.As(style.Panel),
			style.Round(style.RadiusFull),
			style.PadInline(style.Space2),
			style.FontSize(style.TextXs),
			style.FontWeight(style.WeightMedium),
		).
		Part(PartAssignToggle,
			style.As(style.Panel),
			style.Round(style.RadiusFull),
			style.PadInline(style.Space3),
			style.FontSize(style.TextSm),
		).
		Part(PartAssignInput,
			style.As(style.Field),
			style.Round(style.RadiusMd),
			style.PadInline(style.Space3),
			style.ControlBox(),
		).
		Part(PartAssignBtn,
			style.Button(style.Primary),
		).
		Part(PartAssignError,
			style.As(style.DangerWash),
			style.Pad(style.Space2),
			style.Round(style.RadiusSm),
			style.FontSize(style.TextSm),
		).
		Stylesheet()
}
