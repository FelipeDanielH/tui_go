package sim

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

type Engine struct {
	Config    Config
	Seed      int64
	Tick      int64
	Metrics   Metrics
	terrain   []Terrain
	entities  map[int]*Entity
	order     []int
	spatial   map[Point][]int
	nextID    int
	rng       *rand.Rand
	events    []string
	pathSeen  []int
	pathPrev  []int
	pathMark  int
	pathQueue []int
}

func New(seed int64, cfg Config) *Engine {
	if cfg.Width < 16 {
		cfg.Width = 16
	}
	if cfg.Height < 10 {
		cfg.Height = 10
	}
	e := &Engine{
		Config: cfg, Seed: seed, terrain: make([]Terrain, cfg.Width*cfg.Height),
		entities: make(map[int]*Entity), spatial: make(map[Point][]int),
		nextID: 1, rng: rand.New(rand.NewSource(seed)),
		pathSeen: make([]int, cfg.Width*cfg.Height), pathPrev: make([]int, cfg.Width*cfg.Height),
	}
	e.generateTerrain()
	for i := 0; i < cfg.InitialPlants; i++ {
		e.spawnRandom(Plant)
	}
	for i := 0; i < cfg.InitialRabbits; i++ {
		e.spawnRandom(Rabbit)
	}
	for i := 0; i < cfg.InitialWolves; i++ {
		e.spawnRandom(Wolf)
	}
	e.rebuildSpatial()
	return e
}

func (e *Engine) Width() int  { return e.Config.Width }
func (e *Engine) Height() int { return e.Config.Height }

func (e *Engine) InBounds(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < e.Config.Width && p.Y < e.Config.Height
}

func (e *Engine) TerrainAt(p Point) Terrain {
	if !e.InBounds(p) {
		return Water
	}
	return e.terrain[p.Y*e.Config.Width+p.X]
}

func (e *Engine) SetTerrain(p Point, t Terrain) {
	if e.InBounds(p) {
		e.terrain[p.Y*e.Config.Width+p.X] = t
	}
}

func (e *Engine) Entity(id int) *Entity { return e.entities[id] }

func (e *Engine) Entities() []*Entity {
	result := make([]*Entity, 0, len(e.entities))
	for _, id := range e.order {
		if ent := e.entities[id]; ent != nil && ent.Alive {
			result = append(result, ent)
		}
	}
	return result
}

func (e *Engine) EntitiesAt(p Point) []*Entity {
	ids := e.spatial[p]
	result := make([]*Entity, 0, len(ids))
	for _, id := range ids {
		if ent := e.entities[id]; ent != nil && ent.Alive {
			result = append(result, ent)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return renderPriority(result[i].Kind) > renderPriority(result[j].Kind)
	})
	return result
}

func (e *Engine) EntityAt(p Point) *Entity {
	items := e.EntitiesAt(p)
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func renderPriority(k Kind) int {
	switch k {
	case Wolf:
		return 3
	case Rabbit:
		return 2
	default:
		return 1
	}
}

func (e *Engine) Population(k Kind) int {
	n := 0
	for _, ent := range e.entities {
		if ent.Alive && ent.Kind == k {
			n++
		}
	}
	return n
}

func (e *Engine) AddEntity(k Kind, p Point) *Entity {
	if !e.InBounds(p) || e.TerrainAt(p) == Water {
		return nil
	}
	ent := &Entity{ID: e.nextID, Kind: k, Pos: p, Alive: true, TargetID: -1, Needs: Needs{Health: 100, Energy: 80}}
	switch k {
	case Plant:
		ent.Needs.Health = 35 + e.rng.Float64()*45
		ent.Needs.Energy = 100
		ent.Traits = Traits{Size: 0.6, Reach: 0}
	case Rabbit:
		ent.Needs.Hunger = 10 + e.rng.Float64()*25
		ent.Needs.Thirst = 10 + e.rng.Float64()*20
		ent.Traits = Traits{Size: 1, Perception: 9, Speed: 2, Reach: 1}
		ent.Mind.Personality = e.newPersonality()
	case Wolf:
		ent.Needs.Hunger = 15 + e.rng.Float64()*20
		ent.Needs.Thirst = 10 + e.rng.Float64()*20
		ent.Traits = Traits{Size: 1.7, Perception: 12, Speed: 2, Reach: 1}
		ent.Mind.Personality = e.newPersonality()
	}
	e.nextID++
	e.entities[ent.ID] = ent
	e.order = append(e.order, ent.ID)
	e.spatial[p] = append(e.spatial[p], ent.ID)
	return ent
}

func (e *Engine) RemoveEntity(id int) {
	if ent := e.entities[id]; ent != nil {
		e.removeFromSpatial(ent.Pos, id)
		ent.Alive = false
		delete(e.entities, id)
	}
}

func (e *Engine) relocate(ent *Entity, to Point) {
	if ent.Pos == to {
		return
	}
	e.removeFromSpatial(ent.Pos, ent.ID)
	ent.Pos = to
	e.spatial[to] = append(e.spatial[to], ent.ID)
}

func (e *Engine) removeFromSpatial(p Point, id int) {
	ids := e.spatial[p]
	for i, candidate := range ids {
		if candidate != id {
			continue
		}
		ids[i] = ids[len(ids)-1]
		ids = ids[:len(ids)-1]
		if len(ids) == 0 {
			delete(e.spatial, p)
		} else {
			e.spatial[p] = ids
		}
		return
	}
}

func (e *Engine) RecentEvents() []string {
	result := make([]string, len(e.events))
	copy(result, e.events)
	return result
}

func (e *Engine) addEvent(format string, args ...any) {
	e.events = append(e.events, fmt.Sprintf(format, args...))
	if len(e.events) > 8 {
		e.events = e.events[len(e.events)-8:]
	}
}

func (e *Engine) StateDigest() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d|", e.Tick)
	for _, id := range e.order {
		ent := e.entities[id]
		if ent == nil || !ent.Alive {
			continue
		}
		fmt.Fprintf(&b, "%d:%d:%d,%d:%d:%d:%d:%d:%d:%d:%d:%d|", ent.ID, ent.Kind, ent.Pos.X, ent.Pos.Y, ent.Age,
			int(ent.Needs.Health*10), int(ent.Needs.Hunger*10), int(ent.Needs.Thirst*10),
			int(ent.Mind.Personality.Boldness*100), int(ent.Mind.Personality.Curiosity*100), int(ent.Mind.Goal.Kind), len(ent.Mind.Memory.Entries))
		for _, memory := range ent.Mind.Memory.Entries {
			fmt.Fprintf(&b, "m%d:%d,%d:%d:%d:%d|", memory.Kind, memory.Position.X, memory.Position.Y, memory.EntityID, memory.ObservedAt, int(memory.Confidence*100))
		}
		nav := ent.Mind.Navigation
		fmt.Fprintf(&b, "n%d,%d:%d:%d|", nav.Destination.X, nav.Destination.Y, nav.Next, len(nav.Path))
		for _, point := range nav.Path {
			fmt.Fprintf(&b, "p%d,%d|", point.X, point.Y)
		}
	}
	return b.String()
}

func (e *Engine) generateTerrain() {
	// A few deterministic irregular lakes produce readable terrain without
	// committing the project to a complex world generator yet.
	lakes := 3
	for i := 0; i < lakes; i++ {
		cx := 6 + e.rng.Intn(max(1, e.Config.Width-12))
		cy := 4 + e.rng.Intn(max(1, e.Config.Height-8))
		rx := 3 + e.rng.Intn(6)
		ry := 2 + e.rng.Intn(4)
		for y := cy - ry; y <= cy+ry; y++ {
			for x := cx - rx; x <= cx+rx; x++ {
				p := Point{x, y}
				if !e.InBounds(p) {
					continue
				}
				dx, dy := x-cx, y-cy
				jitter := e.rng.Intn(5) - 2
				if dx*dx*ry*ry+dy*dy*rx*rx <= rx*rx*ry*ry+jitter*rx {
					e.SetTerrain(p, Water)
				}
			}
		}
	}
}

func (e *Engine) spawnRandom(k Kind) *Entity {
	for attempts := 0; attempts < 300; attempts++ {
		p := Point{e.rng.Intn(e.Config.Width), e.rng.Intn(e.Config.Height)}
		if e.TerrainAt(p) == Water || (k == Plant && e.plantAt(p) != nil) {
			continue
		}
		ent := e.AddEntity(k, p)
		if ent != nil && ent.Animal() {
			ent.Age = e.rng.Intn(180)
		}
		return ent
	}
	return nil
}

func (e *Engine) plantAt(p Point) *Entity {
	for _, id := range e.spatial[p] {
		if ent := e.entities[id]; ent != nil && ent.Alive && ent.Kind == Plant {
			return ent
		}
	}
	return nil
}

func (e *Engine) rebuildSpatial() {
	e.spatial = make(map[Point][]int, len(e.entities))
	for _, id := range e.order {
		if ent := e.entities[id]; ent != nil && ent.Alive {
			e.spatial[ent.Pos] = append(e.spatial[ent.Pos], id)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
