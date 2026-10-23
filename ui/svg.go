//go:build !wasm

package ui

import "webtyp.com/svg/sprite"

// Icons declares the SVGs for code generation / site extraction.
type Icons struct{}

func (m *Icons) IconSvg() *sprite.Sprite {
	return sprite.NewSprite(
		sprite.Define(IconRoles, "0 0 16 16",
			sprite.Path("M8 1a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM3 13c0-2.5 2.5-4 5-4s5 1.5 5 4v1H3v-1z"),
		),
		sprite.Define(IconAssignments, "0 0 16 16",
			sprite.Path("M11 7V5a3 3 0 1 0-6 0v2H4v7h8V7h-1zm-4-2a1 1 0 1 1 2 0v2H7V5z"),
		),
		sprite.Define(IconAudit, "0 0 16 16",
			sprite.Path("M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zm0 2a5 5 0 1 1 0 10A5 5 0 0 1 8 3zm-.5 2v3.5l3 1.5.5-.8-2.5-1.2V5h-1z"),
		),
	)
}
