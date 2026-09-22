package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/FelipeDanielH/tui_go/internal/sim"
)

type tickMsg time.Time

type Model struct {
	engine       *sim.Engine
	width        int
	height       int
	camera       sim.Point
	cursor       sim.Point
	paused       bool
	speed        int
	showHelp     bool
	lastCycle    int
	selected     int
	inspectorTab int
	debug        bool
	showEco      bool
}

func New(engine *sim.Engine) Model {
	return Model{engine: engine, width: 80, height: 24, speed: 1,
		cursor: sim.Point{X: engine.Width() / 2, Y: engine.Height() / 2}}
}

func (m Model) Init() tea.Cmd { return scheduleTick() }

func scheduleTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureCamera()
	case tickMsg:
		if !m.paused && !m.showHelp {
			for i := 0; i < m.speed; i++ {
				m.engine.Step()
			}
		}
		if ent := m.engine.Entity(m.selected); ent != nil {
			m.cursor = ent.Pos
			m.ensureCamera()
		} else {
			m.selected = 0
		}
		return m, scheduleTick()
	case tea.KeyPressMsg:
		key := msg.String()
		if m.showHelp {
			if key == "?" || key == "esc" || key == "enter" || key == "q" {
				m.showHelp = false
			}
			return m, nil
		}
		switch key {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "?":
			m.showHelp = true
		case "e":
			m.showEco = !m.showEco
		case " ", "space":
			m.paused = !m.paused
		case "+", "=":
			if m.speed < 8 {
				m.speed *= 2
			}
		case "-", "_":
			if m.speed > 1 {
				m.speed /= 2
			}
		case "1":
			m.speed = 1
		case "2":
			m.speed = 2
		case "3":
			m.speed = 4
		case "4":
			m.speed = 8
		case "enter", "i":
			m.toggleSelection()
		case "c":
			m.inspectorTab = (m.inspectorTab + 1) % 2
		case "v":
			m.debug = !m.debug
		case "w", "k", "up":
			m.moveCursor(0, -1)
		case "s", "j", "down":
			m.moveCursor(0, 1)
		case "a", "h", "left":
			m.moveCursor(-1, 0)
		case "d", "l", "right":
			m.moveCursor(1, 0)
		case "tab":
			m.cycleAnimal()
		}
	}
	return m, nil
}

func (m *Model) moveCursor(dx, dy int) {
	m.selected = 0
	m.cursor.X = clampInt(m.cursor.X+dx, 0, m.engine.Width()-1)
	m.cursor.Y = clampInt(m.cursor.Y+dy, 0, m.engine.Height()-1)
	m.ensureCamera()
}

func (m *Model) cycleAnimal() {
	entities := m.engine.Entities()
	for i := 0; i < len(entities); i++ {
		m.lastCycle = (m.lastCycle + 1) % len(entities)
		ent := entities[m.lastCycle]
		if ent.Animal() {
			m.cursor = ent.Pos
			m.selected = ent.ID
			m.ensureCamera()
			return
		}
	}
}

func (m *Model) toggleSelection() {
	if m.selected != 0 {
		m.selected = 0
		return
	}
	if ent := m.engine.EntityAt(m.cursor); ent != nil {
		m.selected = ent.ID
	}
}

func (m Model) inspectedEntity() *sim.Entity {
	if m.selected != 0 {
		return m.engine.Entity(m.selected)
	}
	return m.engine.EntityAt(m.cursor)
}

func (m *Model) viewportSize() (int, int) {
	panel := 0
	if m.width >= 88 {
		panel = 30
	}
	cols := maxInt(10, (m.width-panel)/2)
	rows := maxInt(4, m.height-3)
	return cols, rows
}

func (m *Model) ensureCamera() {
	cols, rows := m.viewportSize()
	marginX, marginY := maxInt(2, cols/5), maxInt(1, rows/5)
	if m.cursor.X < m.camera.X+marginX {
		m.camera.X = m.cursor.X - marginX
	}
	if m.cursor.X >= m.camera.X+cols-marginX {
		m.camera.X = m.cursor.X - cols + marginX + 1
	}
	if m.cursor.Y < m.camera.Y+marginY {
		m.camera.Y = m.cursor.Y - marginY
	}
	if m.cursor.Y >= m.camera.Y+rows-marginY {
		m.camera.Y = m.cursor.Y - rows + marginY + 1
	}
	m.camera.X = clampInt(m.camera.X, 0, maxInt(0, m.engine.Width()-cols))
	m.camera.Y = clampInt(m.camera.Y, 0, maxInt(0, m.engine.Height()-rows))
}

func clampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
