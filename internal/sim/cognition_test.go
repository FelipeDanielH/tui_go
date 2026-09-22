package sim

import "testing"

func clearTerrain(e *Engine) {
	for y := 0; y < e.Height(); y++ {
		for x := 0; x < e.Width(); x++ {
			e.SetTerrain(Point{x, y}, Land)
		}
	}
}

func hasMemory(actor *Entity, kind MemoryKind) bool {
	for _, memory := range actor.Mind.Memory.Entries {
		if memory.Kind == kind {
			return true
		}
	}
	return false
}

func TestMemoriesOnlyComeFromPerception(t *testing.T) {
	e := New(17, emptyConfig())
	clearTerrain(e)
	rabbit := e.AddEntity(Rabbit, Point{4, 4})
	plant := e.AddEntity(Plant, Point{6, 4})
	e.SetTerrain(Point{4, 6}, Water)
	if len(rabbit.Mind.Memory.Entries) != 0 {
		t.Fatal("new animal must not receive map knowledge")
	}
	e.observe(rabbit, e.Perceive(rabbit))
	if !hasMemory(rabbit, MemoryFood) || !hasMemory(rabbit, MemoryWater) {
		t.Fatalf("perceived food and water were not remembered: %+v", rabbit.Mind.Memory.Entries)
	}
	if rabbit.Mind.Memory.Entries[0].ObservedAt != e.Tick || plant.ID == 0 {
		t.Fatal("memory lacks legitimate observation metadata")
	}
}

func TestMemoryExpiresAndStaysBounded(t *testing.T) {
	e := New(18, emptyConfig())
	rabbit := e.AddEntity(Rabbit, landPatch(t, e))
	for i := 0; i < maxMemories+5; i++ {
		e.remember(rabbit, MemoryEntry{Kind: MemoryWater, Position: Point{i, 0}, ObservedAt: int64(i), Confidence: 1})
	}
	if len(rabbit.Mind.Memory.Entries) > maxMemories {
		t.Fatalf("memory grew beyond bound: %d", len(rabbit.Mind.Memory.Entries))
	}
	e.Tick = 300
	e.ageMemories(rabbit)
	if len(rabbit.Mind.Memory.Entries) != 0 {
		t.Fatalf("expired memories remain: %+v", rabbit.Mind.Memory.Entries)
	}
}

func TestGoalPersistsThenInvalidates(t *testing.T) {
	e := New(19, emptyConfig())
	clearTerrain(e)
	rabbit := e.AddEntity(Rabbit, Point{4, 4})
	e.remember(rabbit, MemoryEntry{Kind: MemoryWater, Position: Point{10, 4}, ObservedAt: 0, Confidence: 1})
	d := e.ChooseAction(rabbit, Perception{})
	if d.Goal != GoalWater {
		t.Fatalf("expected water goal, got %+v", d)
	}
	e.setGoal(rabbit, d)
	if !e.goalValid(rabbit) {
		t.Fatal("fresh remembered-water goal should persist")
	}
	e.Tick = 200
	e.ageMemories(rabbit)
	if e.goalValid(rabbit) {
		t.Fatal("goal backed only by expired memory should be abandoned")
	}
}

func TestFoodGoalIsAbandonedWhenResourceIsGone(t *testing.T) {
	e := New(191, emptyConfig())
	clearTerrain(e)
	rabbit := e.AddEntity(Rabbit, Point{4, 4})
	plant := e.AddEntity(Plant, Point{5, 4})
	e.remember(rabbit, MemoryEntry{Kind: MemoryFood, Position: plant.Pos, EntityID: plant.ID, ObservedAt: 0, Confidence: 1})
	rabbit.Mind.Goal = Goal{Kind: GoalFood, Destination: plant.Pos, TargetID: plant.ID, SetAt: 0}
	e.RemoveEntity(plant.ID)
	e.relocate(rabbit, plant.Pos)
	if e.goalValid(rabbit) {
		t.Fatal("food goal should be invalid after its resource disappears at destination")
	}
}

func TestPathfindsAroundWaterAndReportsImpossibleRoute(t *testing.T) {
	e := New(20, emptyConfig())
	clearTerrain(e)
	for y := 0; y < 12; y++ {
		if y != 8 {
			e.SetTerrain(Point{7, y}, Water)
		}
	}
	path, ok := e.FindPath(Point{3, 3}, Point{11, 3}, 500)
	if !ok || len(path) == 0 {
		t.Fatal("expected path around lake barrier")
	}
	for _, p := range path {
		if e.TerrainAt(p) == Water {
			t.Fatalf("path crosses water at %s", p)
		}
	}
	for y := 0; y < e.Height(); y++ {
		e.SetTerrain(Point{7, y}, Water)
	}
	if _, ok := e.FindPath(Point{3, 3}, Point{11, 3}, 500); ok {
		t.Fatal("expected impossible route through complete water barrier")
	}
}

func TestNavigationCachesAndInvalidatesPath(t *testing.T) {
	e := New(21, emptyConfig())
	clearTerrain(e)
	rabbit := e.AddEntity(Rabbit, Point{2, 2})
	target := Point{12, 2}
	e.moveWithNavigation(rabbit, target)
	plannedAt := rabbit.Mind.Navigation.PlannedAt
	if len(rabbit.Mind.Navigation.Path) == 0 {
		t.Fatal("route was not planned")
	}
	e.Tick++
	e.moveWithNavigation(rabbit, target)
	if rabbit.Mind.Navigation.PlannedAt != plannedAt {
		t.Fatal("route was recalculated despite same viable destination")
	}
	next := rabbit.Mind.Navigation.Path[rabbit.Mind.Navigation.Next]
	e.SetTerrain(next, Water)
	rabbit.Traits.Speed = 2
	rabbit.Mind.Navigation.MoveBank = 2
	e.moveWithNavigation(rabbit, target)
	if rabbit.Mind.Navigation.Failures == 0 {
		t.Fatal("blocked cached route did not invalidate")
	}
}

func TestRememberedWaterNavigatesToItsBank(t *testing.T) {
	e := New(211, emptyConfig())
	clearTerrain(e)
	rabbit := e.AddEntity(Rabbit, Point{2, 2})
	water := Point{10, 2}
	e.SetTerrain(water, Water)
	bank := e.navigationDestination(rabbit, water)
	if e.TerrainAt(bank) == Water || Distance(bank, water) != 1 {
		t.Fatalf("water destination should resolve to a land bank, got %s", bank)
	}
	if _, ok := e.FindPath(rabbit.Pos, bank, 200); !ok {
		t.Fatal("expected route to remembered water bank")
	}
}

func TestWolfPursuesLastSeenPreyThenAbandons(t *testing.T) {
	e := New(22, emptyConfig())
	clearTerrain(e)
	wolf := e.AddEntity(Wolf, Point{4, 4})
	prey := e.AddEntity(Rabbit, Point{7, 4})
	e.observe(wolf, e.Perceive(wolf))
	e.relocate(prey, Point{20, 12})
	d := e.ChooseAction(wolf, e.Perceive(wolf))
	if d.Goal != GoalHunt || d.Target != (Point{7, 4}) {
		t.Fatalf("wolf should pursue last seen prey, got %+v", d)
	}
	e.setGoal(wolf, d)
	e.Tick = 100
	e.ageMemories(wolf)
	if e.goalValid(wolf) {
		t.Fatal("stale hunt should eventually be abandoned")
	}
}

func TestRabbitFleesAwayFromPerceivedDanger(t *testing.T) {
	e := New(23, emptyConfig())
	clearTerrain(e)
	rabbit := e.AddEntity(Rabbit, Point{8, 8})
	wolf := e.AddEntity(Wolf, Point{7, 8})
	d := e.ChooseAction(rabbit, e.Perceive(rabbit))
	if d.Goal != GoalFlee || Distance(d.Target, wolf.Pos) <= Distance(rabbit.Pos, wolf.Pos) {
		t.Fatalf("rabbit did not choose useful escape: %+v", d)
	}
}

func TestPersonalityAndCognitionAreDeterministic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width, cfg.Height, cfg.InitialPlants, cfg.InitialRabbits, cfg.InitialWolves = 36, 22, 30, 6, 2
	a, b := New(4242, cfg), New(4242, cfg)
	for i := 0; i < 500; i++ {
		a.Step()
		b.Step()
	}
	if a.StateDigest() != b.StateDigest() {
		t.Fatal("cognition changed deterministic replay")
	}
}
