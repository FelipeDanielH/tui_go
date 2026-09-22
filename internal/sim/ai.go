package sim

import "math"

func (e *Engine) Perceive(actor *Entity) Perception {
	p := Perception{FoodDist: 1 << 30, DangerDist: 1 << 30, PreyDist: 1 << 30, WaterDist: 1 << 30}
	radius := actor.Traits.Perception
	for y := actor.Pos.Y - radius; y <= actor.Pos.Y+radius; y++ {
		for x := actor.Pos.X - radius; x <= actor.Pos.X+radius; x++ {
			pos := Point{x, y}
			if !e.InBounds(pos) {
				continue
			}
			d := Distance(actor.Pos, pos)
			if d > radius {
				continue
			}
			if e.TerrainAt(pos) == Water && d < p.WaterDist {
				p.Water, p.WaterDist, p.HasWater = pos, d, true
			}
			for _, id := range e.spatial[pos] {
				other := e.entities[id]
				if other == nil || !other.Alive || other.ID == actor.ID {
					continue
				}
				switch actor.Kind {
				case Rabbit:
					if other.Kind == Plant && d < p.FoodDist {
						p.Food, p.FoodDist = other, d
					}
					if other.Kind == Wolf && d < p.DangerDist {
						p.Predator, p.DangerDist = other, d
					}
				case Wolf:
					if other.Kind == Rabbit && d < p.PreyDist {
						p.Prey, p.PreyDist = other, d
					}
				}
			}
		}
	}
	return p
}

// ChooseAction implements a compact utility AI. Each behavior competes with a
// score derived from needs and local perception, making decisions inspectable
// without hard-coding one long priority chain.
func (e *Engine) ChooseAction(actor *Entity, p Perception) Decision {
	choices := []Decision{{Action: Explore, Utility: 8}}

	if actor.Needs.Energy < 35 {
		choices = append(choices, Decision{Action: Rest, Utility: 42 + (35-actor.Needs.Energy)*1.2})
	}
	if p.HasWater {
		action := SeekWater
		bonus := 0.0
		if p.WaterDist <= actor.Traits.Reach {
			action, bonus = Drink, 35
		}
		choices = append(choices, Decision{Action: action, Target: p.Water, Utility: actor.Needs.Thirst*1.15 + bonus - float64(p.WaterDist)})
	}

	switch actor.Kind {
	case Rabbit:
		if p.Predator != nil {
			choices = append(choices, Decision{Action: Flee, Target: p.Predator.Pos, TargetID: p.Predator.ID,
				Utility: 130 - float64(p.DangerDist)*8})
		}
		if p.Food != nil {
			action, bonus := SeekFood, 0.0
			if p.FoodDist <= actor.Traits.Reach {
				action, bonus = Eat, 32
			}
			choices = append(choices, Decision{Action: action, Target: p.Food.Pos, TargetID: p.Food.ID,
				Utility: actor.Needs.Hunger + bonus - float64(p.FoodDist)})
		}
	case Wolf:
		if p.Prey != nil {
			bonus := 0.0
			if p.PreyDist <= actor.Traits.Reach {
				bonus = 15
			}
			choices = append(choices, Decision{Action: Hunt, Target: p.Prey.Pos, TargetID: p.Prey.ID,
				Utility: -10 + actor.Needs.Hunger*1.2 + bonus - float64(p.PreyDist)})
		}
	}

	best := choices[0]
	for _, choice := range choices[1:] {
		if choice.Utility > best.Utility {
			best = choice
		}
	}
	return best
}

func (e *Engine) execute(actor *Entity, d Decision) {
	actor.Action, actor.Target, actor.TargetID = d.Action, d.Target, d.TargetID
	switch d.Action {
	case Rest:
		actor.Needs.Energy = clamp(actor.Needs.Energy+3.0, 0, 100)
	case Drink:
		actor.Needs.Thirst = clamp(actor.Needs.Thirst-34, 0, 100)
	case Eat:
		e.eatPlant(actor, d.TargetID)
	case Hunt:
		prey := e.entities[d.TargetID]
		if prey != nil && prey.Alive && Distance(actor.Pos, prey.Pos) <= actor.Traits.Reach {
			e.RemoveEntity(prey.ID)
			e.Metrics.Deaths++
			e.Metrics.Hunts++
			actor.Needs.Hunger = clamp(actor.Needs.Hunger-58, 0, 100)
			actor.Needs.Energy = clamp(actor.Needs.Energy+16, 0, 100)
			e.addEvent("Wolf #%d caught Rabbit #%d", actor.ID, prey.ID)
		} else {
			e.moveToward(actor, d.Target)
		}
	case Flee:
		e.moveAway(actor, d.Target)
	case SeekFood, SeekWater:
		e.moveToward(actor, d.Target)
	case Explore:
		e.explore(actor)
	}
}

func (e *Engine) eatPlant(actor *Entity, targetID int) {
	plant := e.entities[targetID]
	if plant == nil || !plant.Alive || plant.Kind != Plant || Distance(actor.Pos, plant.Pos) > actor.Traits.Reach {
		return
	}
	meal := math.Min(34, plant.Needs.Health)
	plant.Needs.Health -= meal
	actor.Needs.Hunger = clamp(actor.Needs.Hunger-meal*1.25, 0, 100)
	actor.Needs.Energy = clamp(actor.Needs.Energy+meal*0.25, 0, 100)
	if plant.Needs.Health <= 0.1 {
		e.RemoveEntity(plant.ID)
	}
}

func (e *Engine) moveToward(actor *Entity, target Point) {
	dx, dy := sign(target.X-actor.Pos.X), sign(target.Y-actor.Pos.Y)
	candidates := []Point{{dx, dy}, {dx, 0}, {0, dy}}
	e.moveFirstValid(actor, candidates)
}

func (e *Engine) moveAway(actor *Entity, danger Point) {
	dx, dy := sign(actor.Pos.X-danger.X), sign(actor.Pos.Y-danger.Y)
	candidates := []Point{{dx, dy}, {dx, 0}, {0, dy}, {-dy, dx}}
	e.moveFirstValid(actor, candidates)
}

func (e *Engine) explore(actor *Entity) {
	directions := []Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}}
	start := e.rng.Intn(len(directions))
	ordered := append(directions[start:], directions[:start]...)
	e.moveFirstValid(actor, ordered)
}

func (e *Engine) moveFirstValid(actor *Entity, deltas []Point) {
	for _, delta := range deltas {
		if delta == (Point{}) {
			continue
		}
		to := actor.Pos.Add(delta)
		if e.InBounds(to) && e.TerrainAt(to) == Land {
			e.relocate(actor, to)
			actor.Needs.Energy = clamp(actor.Needs.Energy-0.45, 0, 100)
			return
		}
	}
}

func sign(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
