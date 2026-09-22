package sim

// FindPath performs a bounded BFS over land cells. Its search workspace belongs
// to Engine and is reused across calls; only the returned, cached route is
// allocated. Fixed neighbor order makes every route deterministic.
func (e *Engine) FindPath(start, goal Point, limit int) ([]Point, bool) {
	if !e.InBounds(start) || !e.InBounds(goal) || e.TerrainAt(start) == Water || e.TerrainAt(goal) == Water {
		return nil, false
	}
	if start == goal {
		return []Point{}, true
	}
	if limit <= 0 {
		limit = 1200
	}
	e.pathMark++
	if e.pathMark == 0 { // practically unreachable, but keeps marks sound.
		for i := range e.pathSeen {
			e.pathSeen[i] = 0
		}
		e.pathMark = 1
	}
	mark := e.pathMark
	startIndex, goalIndex := e.pointIndex(start), e.pointIndex(goal)
	e.pathQueue = append(e.pathQueue[:0], startIndex)
	e.pathSeen[startIndex], e.pathPrev[startIndex] = mark, startIndex
	directions := [...]Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}, {1, -1}, {1, 1}, {-1, 1}, {-1, -1}}
	for head := 0; head < len(e.pathQueue) && len(e.pathQueue) < limit; head++ {
		current := e.pathQueue[head]
		position := e.indexPoint(current)
		for _, delta := range directions {
			nextPoint := position.Add(delta)
			if !e.InBounds(nextPoint) || e.TerrainAt(nextPoint) == Water {
				continue
			}
			next := e.pointIndex(nextPoint)
			if e.pathSeen[next] == mark {
				continue
			}
			e.pathSeen[next], e.pathPrev[next] = mark, current
			if next == goalIndex {
				return e.buildPath(startIndex, goalIndex), true
			}
			e.pathQueue = append(e.pathQueue, next)
		}
	}
	return nil, false
}

func (e *Engine) buildPath(start, goal int) []Point {
	length := 0
	for current := goal; current != start; current = e.pathPrev[current] {
		length++
	}
	path := make([]Point, length)
	for current := goal; current != start; current = e.pathPrev[current] {
		length--
		path[length] = e.indexPoint(current)
	}
	return path
}

func (e *Engine) pointIndex(p Point) int { return p.Y*e.Config.Width + p.X }
func (e *Engine) indexPoint(index int) Point {
	return Point{X: index % e.Config.Width, Y: index / e.Config.Width}
}
