package sim

import "testing"

func emptyConfig() Config {
	return Config{Width: 28, Height: 18, MaxPlants: 100, MaxRabbits: 20, MaxWolves: 10}
}

func landPatch(t *testing.T, e *Engine) Point {
	t.Helper()
	for y := 2; y < e.Height()-2; y++ {
		for x := 2; x < e.Width()-2; x++ {
			p := Point{x, y}
			if e.TerrainAt(p) == Land && e.TerrainAt(Point{x + 1, y}) == Land && e.TerrainAt(Point{x, y + 1}) == Land {
				return p
			}
		}
	}
	t.Fatal("no usable land patch")
	return Point{}
}

func TestDeterministicForSameSeed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width, cfg.Height = 42, 24
	cfg.InitialPlants, cfg.InitialRabbits, cfg.InitialWolves = 70, 10, 3
	a, b := New(424242, cfg), New(424242, cfg)
	for i := 0; i < 240; i++ {
		a.Step()
		b.Step()
	}
	if a.StateDigest() != b.StateDigest() {
		t.Fatal("same seed and actions produced different states")
	}
}

func TestNeedsAdvanceWithoutUI(t *testing.T) {
	e := New(1, emptyConfig())
	rabbit := e.AddEntity(Rabbit, landPatch(t, e))
	rabbit.Needs = Needs{Health: 100, Hunger: 20, Thirst: 20, Energy: 80}
	e.Step()
	if rabbit.Needs.Hunger <= 20 || rabbit.Needs.Thirst <= 20 {
		t.Fatalf("needs did not increase: %+v", rabbit.Needs)
	}
}

func TestPerceptionIsLocalAndFindsNearestFood(t *testing.T) {
	e := New(2, emptyConfig())
	p := landPatch(t, e)
	rabbit := e.AddEntity(Rabbit, p)
	near := e.AddEntity(Plant, Point{p.X + 1, p.Y})
	far := e.AddEntity(Plant, Point{p.X + 8, p.Y})
	e.rebuildSpatial()
	seen := e.Perceive(rabbit)
	if seen.Food == nil || seen.Food.ID != near.ID {
		t.Fatalf("expected nearest plant %d, got %+v", near.ID, seen.Food)
	}
	if seen.Food.ID == far.ID {
		t.Fatal("perception selected an entity outside its radius")
	}
}

func TestUtilityAIChoosesDangerOverFood(t *testing.T) {
	e := New(3, emptyConfig())
	p := landPatch(t, e)
	rabbit := e.AddEntity(Rabbit, p)
	rabbit.Needs.Hunger, rabbit.Needs.Thirst, rabbit.Needs.Energy = 90, 5, 90
	plant := e.AddEntity(Plant, Point{p.X + 1, p.Y})
	wolf := e.AddEntity(Wolf, Point{p.X, p.Y + 1})
	e.rebuildSpatial()
	decision := e.ChooseAction(rabbit, e.Perceive(rabbit))
	if decision.Action != Flee || decision.TargetID != wolf.ID {
		t.Fatalf("expected flee from wolf, got %+v (plant %d)", decision, plant.ID)
	}
}

func TestPlantGrowthAndConsumption(t *testing.T) {
	e := New(4, emptyConfig())
	p := landPatch(t, e)
	plant := e.AddEntity(Plant, Point{p.X + 1, p.Y})
	plant.Needs.Health = 50
	e.Step()
	if plant.Needs.Health <= 50 {
		t.Fatalf("plant did not grow: %.2f", plant.Needs.Health)
	}
	rabbit := e.AddEntity(Rabbit, p)
	rabbit.Needs = Needs{Health: 100, Hunger: 90, Thirst: 0, Energy: 90}
	e.rebuildSpatial()
	before := plant.Needs.Health
	d := e.ChooseAction(rabbit, e.Perceive(rabbit))
	if d.Action != Eat {
		t.Fatalf("hungry rabbit should eat adjacent plant, got %s", d.Action)
	}
	e.execute(rabbit, d)
	if plant.Needs.Health >= before || rabbit.Needs.Hunger >= 90 {
		t.Fatal("consumption did not transfer plant biomass to rabbit")
	}
}

func TestDeathFromUnmetNeeds(t *testing.T) {
	e := New(5, emptyConfig())
	rabbit := e.AddEntity(Rabbit, landPatch(t, e))
	rabbit.Needs = Needs{Health: 0.5, Hunger: 120, Thirst: 120, Energy: 0}
	e.Step()
	if e.Entity(rabbit.ID) != nil {
		t.Fatal("rabbit with exhausted health should be removed")
	}
}

func TestBasicReproduction(t *testing.T) {
	e := New(6, emptyConfig())
	p := landPatch(t, e)
	a := e.AddEntity(Rabbit, p)
	b := e.AddEntity(Rabbit, Point{p.X + 1, p.Y})
	for _, rabbit := range []*Entity{a, b} {
		rabbit.Age = 300
		rabbit.Needs = Needs{Health: 100, Hunger: 5, Thirst: 5, Energy: 100}
	}
	e.rebuildSpatial()
	before := e.Population(Rabbit)
	child := e.tryReproduce(a, true)
	if child == nil || e.Population(Rabbit) != before+1 || child.Age != 0 {
		t.Fatalf("expected one newborn, child=%+v population=%d", child, e.Population(Rabbit))
	}
}

func BenchmarkStep(b *testing.B) {
	cfg := DefaultConfig()
	e := New(42, cfg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i > 0 && i%1000 == 0 {
			b.StopTimer()
			e = New(42+int64(i), cfg)
			b.StartTimer()
		}
		e.Step()
	}
}
