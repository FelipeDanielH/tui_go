package sim

func (e *Engine) Step() {
	e.Tick++
	e.growPlants()

	// Freeze the turn order. Newborns and new plants begin acting next tick.
	turn := append([]int(nil), e.order...)
	for _, id := range turn {
		actor := e.entities[id]
		if actor == nil || !actor.Alive || !actor.Animal() {
			continue
		}
		e.updateNeeds(actor)
		if !actor.Alive {
			continue
		}
		e.ageMemories(actor)
		perception := e.Perceive(actor)
		e.observe(actor, perception)
		decision := e.ChooseAction(actor, perception)
		e.execute(actor, decision)
		e.tryReproduce(actor, false)
	}
	e.compactOrder()
}

func (e *Engine) updateNeeds(actor *Entity) {
	actor.Age++
	hungerRate, thirstRate := 0.26, 0.32
	if actor.Kind == Wolf {
		hungerRate, thirstRate = 0.20, 0.29
	}
	actor.Needs.Hunger = clamp(actor.Needs.Hunger+hungerRate, 0, 120)
	actor.Needs.Thirst = clamp(actor.Needs.Thirst+thirstRate, 0, 120)
	actor.Needs.Energy = clamp(actor.Needs.Energy-0.16, 0, 100)
	if actor.Cooldown > 0 {
		actor.Cooldown--
	}
	if actor.Needs.Hunger > 92 {
		actor.Needs.Health -= 0.38
	}
	if actor.Needs.Thirst > 90 {
		actor.Needs.Health -= 0.58
	}
	if actor.Needs.Energy <= 1 {
		actor.Needs.Health -= 0.4
	}
	maxAge := 1800
	if actor.Kind == Wolf {
		maxAge = 2400
	}
	if actor.Age > maxAge {
		actor.Needs.Health -= 0.8
	}
	if actor.Needs.Health <= 0 {
		e.addEvent("%s #%d died", actor.Kind, actor.ID)
		e.Metrics.Deaths++
		e.RemoveEntity(actor.ID)
	}
}

func (e *Engine) growPlants() {
	plants := e.Population(Plant)
	for _, id := range append([]int(nil), e.order...) {
		plant := e.entities[id]
		if plant == nil || !plant.Alive || plant.Kind != Plant {
			continue
		}
		plant.Age++
		plant.Needs.Health = clamp(plant.Needs.Health+0.32, 0, 100)
		if plants >= e.Config.MaxPlants || plant.Needs.Health < 62 || e.rng.Float64() >= 0.004 {
			continue
		}
		directions := []Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}}
		start := e.rng.Intn(len(directions))
		for i := 0; i < len(directions); i++ {
			p := plant.Pos.Add(directions[(start+i)%len(directions)])
			if e.InBounds(p) && e.TerrainAt(p) == Land && e.plantAt(p) == nil {
				child := e.AddEntity(Plant, p)
				if child != nil {
					child.Needs.Health = 18
					plant.Needs.Health -= 12
					plants++
				}
				break
			}
		}
	}
}

func (e *Engine) tryReproduce(parent *Entity, force bool) *Entity {
	if parent == nil || !parent.Alive || !parent.Animal() || parent.Cooldown > 0 ||
		parent.Needs.Energy < 60 || parent.Needs.Hunger > 55 || parent.Needs.Thirst > 55 {
		return nil
	}
	maturity, chance, cooldown, cap := 120, 0.018, 150, e.Config.MaxRabbits
	if parent.Kind == Wolf {
		maturity, chance, cooldown, cap = 190, 0.006, 240, e.Config.MaxWolves
	}
	if parent.Age < maturity || e.Population(parent.Kind) >= cap || (!force && e.rng.Float64() >= chance) {
		return nil
	}
	var mate *Entity
	for y := parent.Pos.Y - 2; y <= parent.Pos.Y+2 && mate == nil; y++ {
		for x := parent.Pos.X - 2; x <= parent.Pos.X+2 && mate == nil; x++ {
			for _, id := range e.spatial[Point{x, y}] {
				candidate := e.entities[id]
				if candidate != nil && candidate.Alive && candidate.ID != parent.ID && candidate.Kind == parent.Kind && candidate.Age >= maturity {
					mate = candidate
					break
				}
			}
		}
	}
	if mate == nil {
		return nil
	}
	directions := []Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}}
	start := e.rng.Intn(len(directions))
	for i := 0; i < len(directions); i++ {
		p := parent.Pos.Add(directions[(start+i)%len(directions)])
		if !e.InBounds(p) || e.TerrainAt(p) == Water {
			continue
		}
		child := e.AddEntity(parent.Kind, p)
		if child == nil {
			continue
		}
		child.Age = 0
		child.Needs = Needs{Health: 100, Hunger: 12, Thirst: 12, Energy: 72}
		parent.Needs.Energy -= 24
		mate.Needs.Energy = clamp(mate.Needs.Energy-10, 0, 100)
		parent.Cooldown, mate.Cooldown = cooldown, cooldown/2
		e.Metrics.Births++
		e.addEvent("%s #%d was born", child.Kind, child.ID)
		return child
	}
	return nil
}

func (e *Engine) compactOrder() {
	if len(e.order) < len(e.entities)*2+64 {
		return
	}
	order := make([]int, 0, len(e.entities))
	for _, id := range e.order {
		if ent := e.entities[id]; ent != nil && ent.Alive {
			order = append(order, id)
		}
	}
	e.order = order
}
