package sim

func (e *Engine) newGenome(kind Kind) Genome {
	base := Genome{Size: 1, Speed: 1, Perception: 1, Metabolism: 1, Endurance: 1, Fertility: 1}
	if kind == Wolf {
		base.Size, base.Perception = 1.35, 1.15
	}
	return mutateGenome(base, e.rng, .12)
}

func inheritGenome(a, b Genome, rngFloat func() float64) Genome {
	mix := func(x, y float64) float64 { return x*.35 + y*.35 + (x-y)*(rngFloat()-.5)*.3 }
	return Genome{Size: mix(a.Size, b.Size), Speed: mix(a.Speed, b.Speed), Perception: mix(a.Perception, b.Perception), Metabolism: mix(a.Metabolism, b.Metabolism), Endurance: mix(a.Endurance, b.Endurance), Fertility: mix(a.Fertility, b.Fertility)}
}

func mutateGenome(g Genome, rng interface{ Float64() float64 }, amount float64) Genome {
	mut := func(v float64) float64 { return clamp(v+(rng.Float64()-.5)*amount, .65, 1.45) }
	g.Size, g.Speed, g.Perception, g.Metabolism, g.Endurance, g.Fertility = mut(g.Size), mut(g.Speed), mut(g.Perception), mut(g.Metabolism), mut(g.Endurance), mut(g.Fertility)
	return g
}

func phenotype(kind Kind, g Genome) Phenotype {
	p := Phenotype{Metabolism: g.Metabolism * (.82 + g.Size*.18 + g.Perception*.05), MoveCost: g.Speed * (.70 + g.Size*.25), FoodEfficiency: g.Endurance / (.75 + g.Size*.25), Fertility: g.Fertility, MaxAge: int(1800 * g.Endurance / (.8 + g.Metabolism*.2))}
	if kind == Wolf {
		p.MaxAge = int(2400 * g.Endurance / (.8 + g.Metabolism*.2))
	}
	return p
}

func (e *Engine) applyBiology(ent *Entity) {
	ent.Biology.Phenotype = phenotype(ent.Kind, ent.Biology.Genome)
	ent.Traits.Size = ent.Biology.Genome.Size
	ent.Traits.Perception = max(3, int(float64(ent.Traits.Perception)*ent.Biology.Genome.Perception))
	ent.Traits.Speed = max(1, int(2*ent.Biology.Genome.Speed))
}

func inheritPersonality(a, b Personality, r func() float64) Personality {
	mix := func(x, y float64) float64 { return clamp((x+y)*.38+(r()*.24), .2, .8) }
	return Personality{Boldness: mix(a.Boldness, b.Boldness), Curiosity: mix(a.Curiosity, b.Curiosity), Caution: mix(a.Caution, b.Caution), Persistence: mix(a.Persistence, b.Persistence)}
}
