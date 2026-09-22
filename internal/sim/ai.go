package sim

import (
	"math"
	"sort"
)

func (e *Engine) Perceive(actor *Entity) Perception {
	p := Perception{FoodDist: 1 << 30, DangerDist: 1 << 30, PreyDist: 1 << 30, WaterDist: 1 << 30}
	radius := actor.Traits.Perception
	for y := actor.Pos.Y - radius; y <= actor.Pos.Y+radius; y++ {
		for x := actor.Pos.X - radius; x <= actor.Pos.X+radius; x++ {
			pos := Point{x, y}
			if !e.InBounds(pos) || Distance(actor.Pos, pos) > radius {
				continue
			}
			d := Distance(actor.Pos, pos)
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

// ChooseAction scores only locally perceived facts and bounded memories. The
// best candidates are retained for the inspector after the decision is made.
func (e *Engine) ChooseAction(actor *Entity, p Perception) Decision {
	var storage [6]Decision
	choices := storage[:0]
	choices = append(choices, Decision{Action: Explore, Goal: GoalExplore, Target: e.explorationDestination(actor), Utility: 8 + actor.Mind.Personality.Curiosity*10})
	if actor.Needs.Energy < 35 {
		choices = append(choices, Decision{Action: Rest, Goal: GoalRest, Target: actor.Pos, Utility: 42 + (35-actor.Needs.Energy)*1.2})
	}
	if p.HasWater {
		action, bonus := SeekWater, 16.0
		if p.WaterDist <= actor.Traits.Reach {
			action, bonus = Drink, 35
		}
		choices = append(choices, Decision{Action: action, Goal: GoalWater, Target: p.Water, Utility: actor.Needs.Thirst*1.15 + bonus - float64(p.WaterDist)})
	} else if memory, ok := e.bestMemory(actor, MemoryWater); ok {
		choices = append(choices, Decision{Action: SeekWater, Goal: GoalWater, Target: memory.Position, Utility: actor.Needs.Thirst*1.05 + memory.Confidence*18 - float64(Distance(actor.Pos, memory.Position))})
	}

	switch actor.Kind {
	case Rabbit:
		if p.Predator != nil {
			choices = append(choices, Decision{Action: Flee, Goal: GoalFlee, Target: e.fleeDestination(actor, p.Predator.Pos), TargetID: p.Predator.ID, Utility: 125 + actor.Mind.Personality.Caution*20 - float64(p.DangerDist)*7})
		} else if memory, ok := e.bestMemory(actor, MemoryDanger); ok {
			choices = append(choices, Decision{Action: Flee, Goal: GoalFlee, Target: e.fleeDestination(actor, memory.Position), TargetID: memory.EntityID, Utility: 28 + actor.Mind.Personality.Caution*28 + memory.Confidence*28})
		}
		if p.Food != nil {
			action, bonus := SeekFood, 12.0
			if p.FoodDist <= actor.Traits.Reach {
				action, bonus = Eat, 32
			}
			choices = append(choices, Decision{Action: action, Goal: GoalFood, Target: p.Food.Pos, TargetID: p.Food.ID, Utility: actor.Needs.Hunger + bonus - float64(p.FoodDist)})
		} else if memory, ok := e.bestMemory(actor, MemoryFood); ok {
			choices = append(choices, Decision{Action: SeekFood, Goal: GoalFood, Target: memory.Position, TargetID: memory.EntityID, Utility: actor.Needs.Hunger + memory.Confidence*16 - float64(Distance(actor.Pos, memory.Position))})
		}
	case Wolf:
		if p.Prey != nil {
			bonus := 0.0
			if p.PreyDist <= actor.Traits.Reach {
				bonus = 15
			}
			choices = append(choices, Decision{Action: Hunt, Goal: GoalHunt, Target: p.Prey.Pos, TargetID: p.Prey.ID, Utility: -10 + actor.Needs.Hunger*1.2 + bonus - float64(p.PreyDist) + actor.Mind.Personality.Boldness*6})
		} else if memory, ok := e.bestMemory(actor, MemoryPrey); ok {
			choices = append(choices, Decision{Action: Hunt, Goal: GoalHunt, Target: memory.Position, TargetID: memory.EntityID, Utility: actor.Needs.Hunger*1.05 + memory.Confidence*18 - float64(Distance(actor.Pos, memory.Position))})
		}
	}

	sort.SliceStable(choices, func(i, j int) bool { return choices[i].Utility > choices[j].Utility })
	actor.Mind.Utilities = actor.Mind.Utilities[:0]
	for i := 0; i < len(choices) && i < 4; i++ {
		actor.Mind.Utilities = append(actor.Mind.Utilities, UtilityScore{Action: choices[i].Action, Utility: choices[i].Utility})
	}
	best := choices[0]
	if current, ok := e.currentGoalChoice(actor, choices); ok && current.Utility+actor.Mind.Personality.Persistence*12 >= best.Utility {
		best = current
	}
	return best
}

func (e *Engine) currentGoalChoice(actor *Entity, choices []Decision) (Decision, bool) {
	if !e.goalValid(actor) {
		return Decision{}, false
	}
	for _, candidate := range choices {
		if candidate.Goal == actor.Mind.Goal.Kind {
			return candidate, true
		}
	}
	goal := actor.Mind.Goal
	return Decision{Action: actionForGoal(goal.Kind), Goal: goal.Kind, Target: goal.Destination, TargetID: goal.TargetID}, true
}

func actionForGoal(goal GoalKind) Action {
	switch goal {
	case GoalFood:
		return SeekFood
	case GoalWater:
		return SeekWater
	case GoalFlee:
		return Flee
	case GoalHunt:
		return Hunt
	case GoalRest:
		return Rest
	}
	return Explore
}

func (e *Engine) goalValid(actor *Entity) bool {
	goal := actor.Mind.Goal
	if goal.Kind == GoalNone {
		return false
	}
	age := e.Tick - goal.SetAt
	switch goal.Kind {
	case GoalExplore:
		return age < 80 && actor.Pos != goal.Destination
	case GoalRest:
		return actor.Needs.Energy < 58 && age < 30
	case GoalFlee:
		return age < int64(10+actor.Mind.Personality.Persistence*18)
	case GoalHunt:
		_, ok := e.bestMemory(actor, MemoryPrey)
		return ok && age < 72 && actor.Pos != goal.Destination
	case GoalFood:
		_, ok := e.bestMemory(actor, MemoryFood)
		return ok && (actor.Pos != goal.Destination || e.plantAt(goal.Destination) != nil)
	case GoalWater:
		_, ok := e.bestMemory(actor, MemoryWater)
		return ok && actor.Pos != goal.Destination
	}
	return false
}

func (e *Engine) setGoal(actor *Entity, decision Decision) {
	if decision.Goal == GoalExplore && decision.Target == actor.Pos {
		decision.Target = e.chooseExplorationDestination(actor)
	}
	goal := &actor.Mind.Goal
	if goal.Kind != decision.Goal || goal.TargetID != decision.TargetID {
		*goal = Goal{Kind: decision.Goal, Destination: decision.Target, TargetID: decision.TargetID, SetAt: e.Tick, LastSeen: e.Tick}
		actor.Mind.Navigation.Path, actor.Mind.Navigation.Next = nil, 0
	} else {
		goal.Destination = decision.Target
	}
	if decision.TargetID > 0 {
		goal.LastSeen = e.Tick
	}
}

func (e *Engine) execute(actor *Entity, d Decision) {
	e.setGoal(actor, d)
	actor.Action, actor.Target, actor.TargetID = d.Action, d.Target, d.TargetID
	switch d.Action {
	case Rest:
		actor.Needs.Energy = clamp(actor.Needs.Energy+3, 0, 100)
	case Drink:
		actor.Needs.Thirst = clamp(actor.Needs.Thirst-34, 0, 100)
		actor.Mind.Goal = Goal{}
	case Eat:
		e.eatPlant(actor, d.TargetID)
		actor.Mind.Goal = Goal{}
	case Hunt:
		prey := e.entities[d.TargetID]
		if prey != nil && prey.Alive && Distance(actor.Pos, prey.Pos) <= actor.Traits.Reach {
			e.RemoveEntity(prey.ID)
			e.Metrics.Deaths++
			e.Metrics.Hunts++
			actor.Needs.Hunger = clamp(actor.Needs.Hunger-58, 0, 100)
			actor.Needs.Energy = clamp(actor.Needs.Energy+16, 0, 100)
			actor.Mind.Goal = Goal{}
			e.addEvent("Wolf #%d caught Rabbit #%d", actor.ID, prey.ID)
		} else {
			e.moveWithNavigation(actor, d.Target)
		}
	case Flee, SeekFood, SeekWater, Explore:
		e.moveWithNavigation(actor, e.navigationDestination(actor, d.Target))
	}
}

func (e *Engine) navigationDestination(actor *Entity, target Point) Point {
	if e.TerrainAt(target) != Water {
		return target
	}
	if current := actor.Mind.Navigation.Destination; e.InBounds(current) && e.TerrainAt(current) == Land && Distance(current, target) == 1 {
		return current
	}
	// Water is a resource, not a walkable destination. Reach its closest bank.
	best, bestDistance := actor.Pos, 1<<30
	for _, delta := range [...]Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}, {1, -1}, {1, 1}, {-1, 1}, {-1, -1}} {
		bank := target.Add(delta)
		if !e.InBounds(bank) || e.TerrainAt(bank) == Water {
			continue
		}
		if distance := Distance(actor.Pos, bank); distance < bestDistance {
			best, bestDistance = bank, distance
		}
	}
	return best
}

func (e *Engine) eatPlant(actor *Entity, targetID int) {
	plant := e.entities[targetID]
	if plant == nil || !plant.Alive || plant.Kind != Plant || Distance(actor.Pos, plant.Pos) > actor.Traits.Reach {
		return
	}
	meal := math.Min(34, plant.Needs.Health)
	plant.Needs.Health -= meal
	actor.Needs.Hunger = clamp(actor.Needs.Hunger-meal*1.25, 0, 100)
	actor.Needs.Energy = clamp(actor.Needs.Energy+meal*.25, 0, 100)
	if plant.Needs.Health <= .1 {
		e.RemoveEntity(plant.ID)
	}
}

func (e *Engine) moveWithNavigation(actor *Entity, target Point) bool {
	if actor.Pos == target {
		return true
	}
	nav := &actor.Mind.Navigation
	replan := nav.Destination != target || nav.Next >= len(nav.Path)
	if replan && (len(nav.Path) == 0 || e.Tick-nav.PlannedAt >= 4) {
		path, ok := e.FindPath(actor.Pos, target, 1200)
		if !ok {
			nav.Path = nil
			nav.Next = 0
			nav.Failures++
			return false
		}
		nav.Path, nav.Next, nav.Destination, nav.PlannedAt = path, 0, target, e.Tick
	}
	if nav.Next >= len(nav.Path) || !e.canMove(actor) {
		return false
	}
	next := nav.Path[nav.Next]
	if e.TerrainAt(next) == Water {
		nav.Path = nil
		nav.Next = 0
		nav.Failures++
		return false
	}
	e.relocate(actor, next)
	nav.Next++
	actor.Needs.Energy = clamp(actor.Needs.Energy-.45/float64(actor.Traits.Speed), 0, 100)
	return true
}

func (e *Engine) canMove(actor *Entity) bool {
	// Speed controls cadence; actions never jump across multiple cells.
	actor.Mind.Navigation.MoveBank += actor.Traits.Speed
	if actor.Mind.Navigation.MoveBank < 2 {
		return false
	}
	actor.Mind.Navigation.MoveBank -= 2
	return true
}

func (e *Engine) explorationDestination(actor *Entity) Point {
	goal := actor.Mind.Goal
	if goal.Kind == GoalExplore && e.goalValid(actor) {
		return goal.Destination
	}
	// The random destination is only sampled when exploration wins, in setGoal.
	// This avoids spending RNG calls (and changing unrelated futures) every tick.
	return actor.Pos
}

func (e *Engine) chooseExplorationDestination(actor *Entity) Point {
	best, bestScore := actor.Pos, -1<<30
	for i := 0; i < 10; i++ {
		candidate := actor.Pos.Add(Point{X: e.rng.Intn(15) - 7, Y: e.rng.Intn(15) - 7})
		if !e.InBounds(candidate) || e.TerrainAt(candidate) == Water {
			continue
		}
		score := Distance(actor.Pos, candidate) * 10
		for _, visited := range actor.Mind.Memory.Visits {
			score += Distance(candidate, visited)
		}
		if score > bestScore {
			best, bestScore = candidate, score
		}
	}
	return best
}

func (e *Engine) fleeDestination(actor *Entity, danger Point) Point {
	best, bestScore := actor.Pos, -1
	for _, delta := range [...]Point{{-3, -3}, {0, -3}, {3, -3}, {-3, 0}, {3, 0}, {-3, 3}, {0, 3}, {3, 3}} {
		candidate := actor.Pos.Add(delta)
		if !e.InBounds(candidate) || e.TerrainAt(candidate) == Water {
			continue
		}
		score := Distance(candidate, danger) * 10
		if score > bestScore {
			best, bestScore = candidate, score
		}
	}
	return best
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
