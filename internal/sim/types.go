// Package sim contains the deterministic ecosystem simulation. It has no
// dependency on the terminal UI and can be used by tests or headless tools.
package sim

import "fmt"

type Point struct{ X, Y int }

func (p Point) Add(q Point) Point { return Point{p.X + q.X, p.Y + q.Y} }

func (p Point) String() string { return fmt.Sprintf("(%d,%d)", p.X, p.Y) }

func Distance(a, b Point) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

type Terrain uint8

const (
	Land Terrain = iota
	Water
)

type Kind uint8

const (
	Plant Kind = iota
	Rabbit
	Wolf
)

func (k Kind) String() string {
	switch k {
	case Plant:
		return "Plant"
	case Rabbit:
		return "Rabbit"
	case Wolf:
		return "Wolf"
	default:
		return "Unknown"
	}
}

type Action uint8

const (
	Idle Action = iota
	Explore
	SeekFood
	Eat
	SeekWater
	Drink
	Rest
	Flee
	Hunt
)

func (a Action) String() string {
	return [...]string{"IDLE", "EXPLORE", "SEEK FOOD", "EAT", "SEEK WATER", "DRINK", "REST", "FLEE", "HUNT"}[a]
}

type Needs struct {
	Health float64
	Hunger float64
	Thirst float64
	Energy float64
}

type Traits struct {
	Size       float64
	Perception int
	Speed      int
	Reach      int
}

type Entity struct {
	ID       int
	Kind     Kind
	Pos      Point
	Age      int
	Needs    Needs
	Traits   Traits
	Action   Action
	TargetID int
	Target   Point
	Cooldown int
	Alive    bool
}

func (e *Entity) Animal() bool { return e.Kind == Rabbit || e.Kind == Wolf }

type Config struct {
	Width          int
	Height         int
	InitialPlants  int
	InitialRabbits int
	InitialWolves  int
	MaxPlants      int
	MaxRabbits     int
	MaxWolves      int
}

type Metrics struct {
	Births int
	Deaths int
	Hunts  int
}

func DefaultConfig() Config {
	return Config{
		Width: 96, Height: 48,
		InitialPlants: 210, InitialRabbits: 22, InitialWolves: 3,
		MaxPlants: 800, MaxRabbits: 90, MaxWolves: 24,
	}
}

type Perception struct {
	Food       *Entity
	FoodDist   int
	Predator   *Entity
	DangerDist int
	Prey       *Entity
	PreyDist   int
	Water      Point
	WaterDist  int
	HasWater   bool
}

type Decision struct {
	Action   Action
	Target   Point
	TargetID int
	Utility  float64
}
