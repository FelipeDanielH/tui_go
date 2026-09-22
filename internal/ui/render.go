package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/FelipeDanielH/tui_go/internal/sim"
)

var (
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#77847A"))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D8D59C")).Bold(true)
	landStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#31533A"))
	waterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4C91A8"))
	plantStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6FAF55"))
	rabbitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E8DFC5")).Bold(true)
	wolfStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#D17B49")).Bold(true)
	cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("#56583F")).Bold(true)
	dangerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75"))
	debugStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#B28DFF"))
)

func (m Model) View() tea.View {
	var content string
	if m.showHelp {
		content = m.renderHelp()
	} else {
		content = m.renderGame()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "TUI Go — Living Terrarium"
	return v
}

func (m Model) renderGame() string {
	status := "RUNNING"
	if m.paused {
		status = "PAUSED"
	}
	headerText := fmt.Sprintf(" TUI GO  %s  ×%d  tick %d  seed %d  P:%d R:%d W:%d",
		status, m.speed, m.engine.Tick, m.engine.Seed,
		m.engine.Population(sim.Plant), m.engine.Population(sim.Rabbit), m.engine.Population(sim.Wolf))
	header := accentStyle.Render(fitPlain(headerText, m.width))

	world := m.renderWorld()
	if m.width >= 88 {
		world = lipgloss.JoinHorizontal(lipgloss.Top, world, m.renderInspector(30))
	}
	footer := mutedStyle.Render(fitPlain(" move WASD/HJKL  enter inspect  tab next  C mind  V debug  space pause  ? help  q quit", m.width))
	return header + "\n" + world + "\n" + footer
}

func (m Model) renderWorld() string {
	cols, rows := m.viewportSize()
	var b strings.Builder
	for sy := 0; sy < rows; sy++ {
		wy := m.camera.Y + sy
		for sx := 0; sx < cols; sx++ {
			wx := m.camera.X + sx
			p := sim.Point{X: wx, Y: wy}
			glyph, style := m.glyphAt(p)
			if debugGlyph, debugStyle, ok := m.debugGlyph(p); ok {
				glyph, style = debugGlyph, debugStyle
			}
			cell := padCell(glyph, 2)
			if p == m.cursor {
				cell = cursorStyle.Render(cell)
			} else {
				cell = style.Render(cell)
			}
			b.WriteString(cell)
		}
		if sy < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) debugGlyph(p sim.Point) (string, lipgloss.Style, bool) {
	if !m.debug || m.selected == 0 {
		return "", lipgloss.Style{}, false
	}
	ent := m.engine.Entity(m.selected)
	if ent == nil || !ent.Animal() || p == ent.Pos {
		return "", lipgloss.Style{}, false
	}
	if p == ent.Mind.Goal.Destination && ent.Mind.Goal.Kind != sim.GoalNone {
		return "◎", debugStyle, true
	}
	for i := ent.Mind.Navigation.Next; i < len(ent.Mind.Navigation.Path); i++ {
		if p == ent.Mind.Navigation.Path[i] {
			return "•", debugStyle, true
		}
	}
	if sim.Distance(p, ent.Pos) == ent.Traits.Perception && m.engine.TerrainAt(p) == sim.Land {
		return "·", debugStyle, true
	}
	return "", lipgloss.Style{}, false
}

func (m Model) glyphAt(p sim.Point) (string, lipgloss.Style) {
	if ent := m.engine.EntityAt(p); ent != nil {
		switch ent.Kind {
		case sim.Wolf:
			return "W", wolfStyle
		case sim.Rabbit:
			return "r", rabbitStyle
		case sim.Plant:
			if ent.Needs.Health > 70 {
				return "♣", plantStyle
			}
			return "♧", plantStyle
		}
	}
	if m.engine.TerrainAt(p) == sim.Water {
		return "≈", waterStyle
	}
	return "·", landStyle
}

func (m Model) renderInspector(width int) string {
	if ent := m.inspectedEntity(); ent != nil && ent.Animal() && m.inspectorTab == 1 {
		return m.renderMindInspector(width, ent)
	}
	inner := width - 2
	lines := []string{"", accentStyle.Render(" OBSERVER"), mutedStyle.Render(fmt.Sprintf(" cursor %s", m.cursor))}
	if ent := m.inspectedEntity(); ent != nil {
		lines = append(lines, "", fmt.Sprintf(" %s #%d", ent.Kind, ent.ID))
		if m.selected == ent.ID {
			lines = append(lines, mutedStyle.Render(" selected · following"))
		}
		if ent.Kind == sim.Plant {
			lines = append(lines, " Biomass "+bar(ent.Needs.Health, inner-10), fmt.Sprintf(" Age     %d ticks", ent.Age))
		} else {
			lines = append(lines,
				" Health "+bar(ent.Needs.Health, inner-9),
				" Hunger "+inverseBar(ent.Needs.Hunger, inner-9),
				" Thirst "+inverseBar(ent.Needs.Thirst, inner-9),
				" Energy "+bar(ent.Needs.Energy, inner-9),
				fmt.Sprintf(" Age     %d ticks", ent.Age),
				fmt.Sprintf(" Action  %s", ent.Action),
			)
			if ent.TargetID > 0 {
				if target := m.engine.Entity(ent.TargetID); target != nil {
					lines = append(lines, fmt.Sprintf(" Target  %s #%d", target.Kind, target.ID))
				} else {
					lines = append(lines, fmt.Sprintf(" Target  %s", ent.Target))
				}
			}
		}
	} else {
		terrain := "land"
		if m.engine.TerrainAt(m.cursor) == sim.Water {
			terrain = "water"
		}
		lines = append(lines, "", mutedStyle.Render(" Empty "+terrain))
	}
	lines = append(lines, "", accentStyle.Render(" RECENT LIFE"))
	events := m.engine.RecentEvents()
	if len(events) == 0 {
		lines = append(lines, mutedStyle.Render(" The terrarium is waking up…"))
	} else {
		for i := len(events) - 1; i >= 0 && len(lines) < m.height-3; i-- {
			lines = append(lines, mutedStyle.Render(" "+events[i]))
		}
	}
	for len(lines) < maxInt(4, m.height-3) {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = fitLine(lines[i], width)
	}
	return strings.Join(lines[:maxInt(4, m.height-3)], "\n")
}

func (m Model) renderMindInspector(width int, ent *sim.Entity) string {
	inner := width - 2
	nav := ent.Mind.Navigation
	goal := ent.Mind.Goal
	lines := []string{"", accentStyle.Render(" MIND · C TO SUMMARY"), mutedStyle.Render(fmt.Sprintf(" %s #%d · V debug:%t", ent.Kind, ent.ID, m.debug)), ""}
	lines = append(lines,
		fmt.Sprintf(" Goal    %s", goal.Kind),
		fmt.Sprintf(" Dest    %s", goal.Destination),
		fmt.Sprintf(" Route   %d steps · %d replans", maxInt(0, len(nav.Path)-nav.Next), nav.Failures),
		fmt.Sprintf(" Traits  B %.2f C %.2f", ent.Mind.Personality.Boldness, ent.Mind.Personality.Caution),
		fmt.Sprintf("         Cur %.2f Per %.2f", ent.Mind.Personality.Curiosity, ent.Mind.Personality.Persistence),
		"", accentStyle.Render(" PERCEPTION"),
		perceptionLine(" water", ent.Mind.Perception.WaterDistance),
		perceptionLine(" food", ent.Mind.Perception.FoodDistance),
		perceptionLine(" danger", ent.Mind.Perception.DangerDistance),
		perceptionLine(" prey", ent.Mind.Perception.PreyDistance),
		"", accentStyle.Render(" TOP UTILITIES"),
	)
	for _, score := range ent.Mind.Utilities {
		lines = append(lines, fmt.Sprintf(" %-10s %5.1f", score.Action, score.Utility))
	}
	lines = append(lines, "", accentStyle.Render(" RECENT MEMORY"))
	for i := len(ent.Mind.Memory.Entries) - 1; i >= 0 && len(lines) < m.height-3; i-- {
		memory := ent.Mind.Memory.Entries[i]
		lines = append(lines, fmt.Sprintf(" %-6s %s  %.0f%%", memory.Kind, memory.Position, memory.Confidence*100))
	}
	for len(lines) < maxInt(4, m.height-3) {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = fitLine(lines[i], inner+2)
	}
	return strings.Join(lines[:maxInt(4, m.height-3)], "\n")
}

func perceptionLine(name string, distance int) string {
	if distance >= 1<<29 {
		return name + "   —"
	}
	return fmt.Sprintf(" %s  %d cells", name, distance)
}

func (m Model) renderHelp() string {
	lines := []string{
		"TUI GO — HELP", "",
		"Observe a self-running ecosystem. No action is required.", "",
		"WASD / HJKL / arrows   Move the world cursor",
		"Enter / I              Select entity and follow it",
		"Tab                    Jump to the next animal",
		"C                      Toggle summary/mind inspector",
		"V                      Toggle selected animal debug overlay",
		"Space                  Pause or resume simulation",
		"+ / -                  Double or halve simulation speed",
		"1 2 3 4                Set speed to ×1, ×2, ×4, ×8",
		"?                      Open or close this help",
		"Q / Esc                Quit (Esc closes help first)", "",
		"Legend", "  ♣ mature vegetation   ♧ young vegetation", "  r rabbit              W wolf              ≈ water", "",
		"The cursor follows a camera over a world larger than the terminal.",
		"Animals only perceive their local area and choose the highest-utility",
		"action from hunger, thirst, energy, danger, and opportunity.", "",
		"Press ? / Enter / Esc to return",
	}
	for i := range lines {
		if i == 0 {
			lines[i] = accentStyle.Render(lines[i])
		} else {
			lines[i] = fitLine(lines[i], maxInt(20, m.width-2))
		}
	}
	return strings.Join(lines, "\n")
}

func bar(value float64, width int) string {
	width = clampInt(width, 4, 18)
	filled := clampInt(int(value/100*float64(width)+0.5), 0, width)
	return plantStyle.Render(strings.Repeat("█", filled)) + mutedStyle.Render(strings.Repeat("░", width-filled))
}

func inverseBar(value float64, width int) string {
	width = clampInt(width, 4, 18)
	filled := clampInt(int(value/100*float64(width)+0.5), 0, width)
	style := accentStyle
	if value > 75 {
		style = dangerStyle
	}
	return style.Render(strings.Repeat("█", filled)) + mutedStyle.Render(strings.Repeat("░", width-filled))
}

func padCell(s string, width int) string {
	for lipgloss.Width(s) > width && len(s) > 0 {
		runes := []rune(s)
		s = string(runes[:len(runes)-1])
	}
	return s + strings.Repeat(" ", maxInt(0, width-lipgloss.Width(s)))
}

func fitLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func fitPlain(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) > width {
		var b strings.Builder
		for _, r := range s {
			candidate := b.String() + string(r)
			if lipgloss.Width(candidate) > width-1 {
				break
			}
			b.WriteRune(r)
		}
		s = b.String() + "…"
	}
	return s + strings.Repeat(" ", maxInt(0, width-lipgloss.Width(s)))
}
