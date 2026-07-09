package theme

import "github.com/charmbracelet/lipgloss"

// Role identifies a semantic colour slot shared across TUI chrome.
type Role int

const (
	RoleBg Role = iota
	RoleFg
	RoleSurface
	RoleOnSurface
	RoleAccent
	RoleOnAccent
	RoleMuted
	RoleSubtle
	RoleBorder
	RoleBorderAccent
	RoleSelectionBg
	RoleSelectionFg
	RoleTitleFg
	RoleTitleBg
	RoleInfo
	RoleSuccess
	RoleWarning
	RoleError
	RoleCritical
	RoleDanger
	RoleDiffAddedBg
	RoleDiffRemovedBg
	numRoles
)

var roleNames = [numRoles]string{
	RoleBg: "bg", RoleFg: "fg",
	RoleSurface: "surface", RoleOnSurface: "on_surface",
	RoleAccent: "accent", RoleOnAccent: "on_accent",
	RoleMuted: "muted", RoleSubtle: "subtle",
	RoleBorder: "border", RoleBorderAccent: "border_accent",
	RoleSelectionBg: "selection_bg", RoleSelectionFg: "selection_fg",
	RoleTitleFg: "title_fg", RoleTitleBg: "title_bg",
	RoleInfo: "info", RoleSuccess: "success", RoleWarning: "warning",
	RoleError: "error", RoleCritical: "critical", RoleDanger: "danger",
	RoleDiffAddedBg: "diff_added_bg", RoleDiffRemovedBg: "diff_removed_bg",
}

func (r Role) String() string {
	if r < 0 || int(r) >= len(roleNames) {
		return ""
	}
	return roleNames[r]
}

// Roles returns every semantic role in declaration order.
func Roles() []Role {
	out := make([]Role, 0, numRoles)
	for r := Role(0); r < numRoles; r++ {
		out = append(out, r)
	}
	return out
}

// Palette is a named table of hex colours per role.
type Palette struct {
	name string
	hex  [numRoles]string
}

func (p *Palette) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

func (p *Palette) Hex(r Role) string {
	if p == nil || r < 0 || int(r) >= len(p.hex) {
		return ""
	}
	return p.hex[r]
}

var darkBase = [numRoles]string{
	RoleBg: "#1A1B26", RoleFg: "#C0CAF5",
	RoleSurface: "#30363d", RoleOnSurface: "#FFFFFF",
	RoleAccent: "#7E00AF", RoleOnAccent: "#FFFFFF",
	RoleMuted: "#6272A4", RoleSubtle: "#6E6E6E",
	RoleBorder: "#9A9A9A", RoleBorderAccent: "#58A6FF",
	RoleSelectionBg: "#3D4F5F", RoleSelectionFg: "#FFFFFF",
	RoleTitleFg: "#BBBBBB", RoleTitleBg: "#2C2C2C",
	RoleInfo: "#7AA2F7", RoleSuccess: "#50FA7B",
	RoleWarning: "#E0AF68", RoleError: "#F7768E", RoleCritical: "#FF5555",
	RoleDanger: "#7A1F1F",
	RoleDiffAddedBg: "#173027", RoleDiffRemovedBg: "#2C1A1F",
}

var lightBase = [numRoles]string{
	RoleBg: "#FFFFFF", RoleFg: "#24292F",
	RoleSurface: "#D0D7DE", RoleOnSurface: "#24292F",
	RoleAccent: "#7E00AF", RoleOnAccent: "#FFFFFF",
	RoleMuted: "#6272A4", RoleSubtle: "#9A9A9A",
	RoleBorder: "#666666", RoleBorderAccent: "#1F6FEB",
	RoleSelectionBg: "#CCE0FF", RoleSelectionFg: "#24292F",
	RoleTitleFg: "#555555", RoleTitleBg: "#E8E8E8",
	RoleInfo: "#1F6FEB", RoleSuccess: "#2A7A2A",
	RoleWarning: "#B7791F", RoleError: "#C4314B", RoleCritical: "#D63031",
	RoleDanger: "#B01515",
	RoleDiffAddedBg: "#E6FFED", RoleDiffRemovedBg: "#FFEEF0",
}

func buildPalette(name string, base [numRoles]string, primary, secondary, muted string) *Palette {
	p := &Palette{name: name, hex: base}
	if primary != "" {
		p.hex[RoleAccent] = normalizeHex(primary)
	}
	if secondary != "" {
		p.hex[RoleSuccess] = normalizeHex(secondary)
	}
	if muted != "" {
		p.hex[RoleMuted] = normalizeHex(muted)
	}
	return p
}

func darkPalette(primary, secondary, muted string) *Palette {
	return buildPalette("dark", darkBase, primary, secondary, muted)
}

func lightPalette(primary, secondary, muted string) *Palette {
	return buildPalette("light", lightBase, primary, secondary, muted)
}

// Adaptive returns role r as a lipgloss.AdaptiveColor using the active theme's
// three configurable colours layered on the dark/light presets.
func Adaptive(r Role) lipgloss.AdaptiveColor {
	currentMu.RLock()
	t := current
	currentMu.RUnlock()
	d := darkPalette(t.Primary, t.Secondary, t.Muted)
	l := lightPalette(t.Primary, t.Secondary, t.Muted)
	return lipgloss.AdaptiveColor{Light: l.Hex(r), Dark: d.Hex(r)}
}
