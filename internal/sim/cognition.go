package sim

const (
	maxMemories = 12
	maxVisits   = 10
)

func (e *Engine) observe(actor *Entity, p Perception) {
	if !actor.Animal() {
		return
	}
	if p.HasWater {
		e.remember(actor, MemoryEntry{Kind: MemoryWater, Position: p.Water, ObservedAt: e.Tick, Confidence: 1})
	}
	if p.Food != nil {
		e.remember(actor, MemoryEntry{Kind: MemoryFood, Position: p.Food.Pos, EntityID: p.Food.ID, ObservedAt: e.Tick, Confidence: 1})
	}
	if p.Prey != nil {
		e.remember(actor, MemoryEntry{Kind: MemoryPrey, Position: p.Prey.Pos, EntityID: p.Prey.ID, ObservedAt: e.Tick, Confidence: 1})
	}
	if p.Predator != nil {
		e.remember(actor, MemoryEntry{Kind: MemoryDanger, Position: p.Predator.Pos, EntityID: p.Predator.ID, ObservedAt: e.Tick, Confidence: 1})
	}
	actor.Mind.Perception = PerceptionSummary{FoodDistance: p.FoodDist, WaterDistance: p.WaterDist, DangerDistance: p.DangerDist, PreyDistance: p.PreyDist}
	e.rememberVisit(actor, actor.Pos)
}

func (e *Engine) remember(actor *Entity, entry MemoryEntry) {
	entries := actor.Mind.Memory.Entries
	for i := range entries {
		current := &entries[i]
		if current.Kind == entry.Kind && ((entry.EntityID > 0 && current.EntityID == entry.EntityID) || (entry.EntityID == 0 && current.Position == entry.Position)) {
			*current = entry
			actor.Mind.Memory.Entries = entries
			return
		}
	}
	if len(entries) >= maxMemories {
		oldest := 0
		for i := 1; i < len(entries); i++ {
			if entries[i].ObservedAt < entries[oldest].ObservedAt {
				oldest = i
			}
		}
		entries[oldest] = entry
	} else {
		entries = append(entries, entry)
	}
	actor.Mind.Memory.Entries = entries
}

func (e *Engine) rememberVisit(actor *Entity, p Point) {
	visits := actor.Mind.Memory.Visits
	if len(visits) > 0 && visits[len(visits)-1] == p {
		return
	}
	visits = append(visits, p)
	if len(visits) > maxVisits {
		visits = visits[len(visits)-maxVisits:]
	}
	actor.Mind.Memory.Visits = visits
}

func (e *Engine) ageMemories(actor *Entity) {
	entries := actor.Mind.Memory.Entries[:0]
	for _, entry := range actor.Mind.Memory.Entries {
		age := e.Tick - entry.ObservedAt
		ttl := int64(180)
		if entry.Kind == MemoryPrey || entry.Kind == MemoryDanger {
			ttl = 72
		}
		if age >= ttl {
			continue
		}
		entry.Confidence = 1 - float64(age)/float64(ttl)
		entries = append(entries, entry)
	}
	actor.Mind.Memory.Entries = entries
}

func (e *Engine) bestMemory(actor *Entity, kind MemoryKind) (MemoryEntry, bool) {
	var best MemoryEntry
	found := false
	bestScore := -1.0
	for _, entry := range actor.Mind.Memory.Entries {
		if entry.Kind != kind {
			continue
		}
		score := entry.Confidence*100 - float64(Distance(actor.Pos, entry.Position))
		if !found || score > bestScore {
			best, bestScore, found = entry, score, true
		}
	}
	return best, found
}

func (e *Engine) newPersonality() Personality {
	// Narrow ranges keep individual variation believable and deterministic.
	return Personality{
		Boldness:    0.35 + e.rng.Float64()*0.30,
		Curiosity:   0.35 + e.rng.Float64()*0.30,
		Caution:     0.35 + e.rng.Float64()*0.30,
		Persistence: 0.35 + e.rng.Float64()*0.30,
	}
}
