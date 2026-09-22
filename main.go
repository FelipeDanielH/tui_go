package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/FelipeDanielH/tui_go/internal/sim"
	"github.com/FelipeDanielH/tui_go/internal/ui"
)

func main() {
	seed := flag.Int64("seed", time.Now().UnixNano(), "world seed (use the same value to replay)")
	worldWidth := flag.Int("width", 96, "world width in cells")
	worldHeight := flag.Int("height", 48, "world height in cells")
	headless := flag.Int("headless", 0, "run N simulation ticks without the TUI")
	flag.Parse()

	cfg := sim.DefaultConfig()
	cfg.Width, cfg.Height = *worldWidth, *worldHeight
	engine := sim.New(*seed, cfg)
	if *headless > 0 {
		for i := 0; i < *headless; i++ {
			engine.Step()
		}
		fmt.Printf("seed=%d tick=%d plants=%d rabbits=%d wolves=%d births=%d deaths=%d hunts=%d\n", engine.Seed, engine.Tick,
			engine.Population(sim.Plant), engine.Population(sim.Rabbit), engine.Population(sim.Wolf),
			engine.Metrics.Births, engine.Metrics.Deaths, engine.Metrics.Hunts)
		return
	}

	program := tea.NewProgram(ui.New(engine))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui_go: %v\n", err)
		os.Exit(1)
	}
}
