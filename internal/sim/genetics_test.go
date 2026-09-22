package sim

import "testing"

func TestBiparentalInheritanceGenerationAndCosts(t *testing.T) {
	e := New(71, emptyConfig())
	clearTerrain(e)
	a := e.AddEntity(Rabbit, Point{4, 4})
	b := e.AddEntity(Rabbit, Point{5, 4})
	a.Age, b.Age = 300, 300
	a.Needs, b.Needs = Needs{Health: 100, Energy: 100}, Needs{Health: 100, Energy: 100}
	a.Biology.Genome.Speed = .8
	b.Biology.Genome.Speed = 1.2
	e.applyBiology(a)
	e.applyBiology(b)
	e.rebuildSpatial()
	child := e.tryReproduce(a, true)
	if child == nil {
		t.Fatal("expected child")
	}
	if child.Biology.Generation != 1 || child.Biology.ParentA != a.ID || child.Biology.ParentB != b.ID {
		t.Fatalf("bad genealogy %+v", child.Biology)
	}
	if child.Biology.Genome.Speed <= .65 || child.Biology.Genome.Speed >= 1.45 || child.Biology.Genome.Speed == a.Biology.Genome.Speed {
		t.Fatalf("unbounded or clone genome %+v", child.Biology.Genome)
	}
	if a.Needs.Energy >= 100 || b.Needs.Energy >= 100 || a.Needs.Hunger == 0 {
		t.Fatal("reproduction lacked ecological cost")
	}
}

func TestGenomeTradeoffsAndDeterminism(t *testing.T) {
	fast := phenotype(Rabbit, Genome{Size: 1, Speed: 1.4, Perception: 1, Metabolism: 1, Endurance: 1, Fertility: 1})
	slow := phenotype(Rabbit, Genome{Size: 1, Speed: .7, Perception: 1, Metabolism: 1, Endurance: 1, Fertility: 1})
	if fast.MoveCost <= slow.MoveCost {
		t.Fatal("speed should cost more movement energy")
	}
	big := phenotype(Rabbit, Genome{Size: 1.4, Speed: 1, Perception: 1, Metabolism: 1, Endurance: 1, Fertility: 1})
	small := phenotype(Rabbit, Genome{Size: .7, Speed: 1, Perception: 1, Metabolism: 1, Endurance: 1, Fertility: 1})
	if big.Metabolism <= small.Metabolism {
		t.Fatal("size should cost metabolism")
	}
	a, b := New(72, DefaultConfig()), New(72, DefaultConfig())
	for i := 0; i < 400; i++ {
		a.Step()
		b.Step()
	}
	if a.StateDigest() != b.StateDigest() {
		t.Fatal("genetics broke determinism")
	}
}

func TestPopulationHistoryIsBounded(t *testing.T) {
	e := New(73, emptyConfig())
	for i := 0; i < 4000; i++ {
		e.Step()
	}
	if len(e.PopulationHistory()) != 64 {
		t.Fatalf("history=%d", len(e.PopulationHistory()))
	}
}
