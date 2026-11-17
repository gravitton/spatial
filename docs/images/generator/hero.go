package main

import (
	"cmp"
	"fmt"
	"slices"

	geom "github.com/gravitton/geometry"
	"github.com/gravitton/spatial"
)

const (
	heroScale  = 19.0
	heroCell   = 3.0
	heroRadius = 6.0
)

var heroMap = []string{
	"...............",
	"..#.....#......",
	"..#.....#....G.",
	"..#..~~.#......",
	"....S~~........",
	"..#..~~...##...",
	"..#.......#....",
	"..........#....",
	"...............",
}

type body struct {
	box  geom.Box[float64]
	kind byte
}

func hero(t theme) []*panel {
	bodies := heroBodies()
	index := spatial.NewHash[int](heroCell)
	for key, b := range bodies {
		index.Set(key, b.box)
	}
	area := geom.Bx(geom.Pt(0.0, 0.0), geom.Pt(float64(len(heroMap[0])), float64(len(heroMap))))
	scout, goal := heroCenter(bodies, 'S'), heroCenter(bodies, 'G')

	view := geom.Bx(scout.AddXY(-heroRadius, -heroRadius), scout.AddXY(heroRadius, heroRadius))
	inView := heroPanel(t, "QueryBox(view)", area, view.IntersectsBox)
	viewKeys := index.QueryBox(view)
	inView.bodies(bodies, viewKeys)
	inView.outline(overlapOf(view, area), style{stroke: t.cool, width: 2.5, dashed: true})
	inView.note = fmt.Sprintf("%d keys", len(viewKeys))

	sight := geom.Seg(scout, goal)
	onSight := heroPanel(t, "QuerySegment(sight)", area, sight.IntersectsBox)
	sightKeys := slices.Collect(index.QuerySegment(sight))
	onSight.bodies(bodies, sightKeys)
	onSight.segment(sight, t.cool, false)
	onSight.note = fmt.Sprintf("%d keys, the scout's own among them", len(sightKeys))

	pairs := index.Pairs()
	slices.SortFunc(pairs, func(a, b spatial.Pair[int]) int {
		return cmp.Or(cmp.Compare(a.A, b.A), cmp.Compare(a.B, b.B))
	})
	var paired []int
	for _, pair := range pairs {
		paired = append(paired, pair.A, pair.B)
	}
	touching := heroPanel(t, "Pairs()", area, nil)
	touching.bodies(bodies, paired)
	for _, pair := range pairs {
		touching.outline(overlapOf(bodies[pair.A].box, bodies[pair.B].box), style{stroke: t.leaf, width: 4})
	}
	touching.note = fmt.Sprintf("%d pairs", len(pairs))

	return []*panel{inView, onSight, touching}
}

func heroBodies() []body {
	var bodies []body
	used := map[geom.Point[int]]bool{}
	for column := range len(heroMap[0]) {
		for row := 0; row < len(heroMap); row++ {
			end := row
			for end < len(heroMap) && heroMap[end][column] == '#' {
				end++
			}
			if end-row > 1 {
				bodies = append(bodies, body{box: heroTiles(column, row, column+1, end), kind: '#'})
				for r := row; r < end; r++ {
					used[geom.Pt(column, r)] = true
				}
			}
			row = max(row, end)
		}
	}

	rough := geom.Box[float64]{}
	for row, line := range heroMap {
		for column, tile := range []byte(line) {
			switch tile {
			case '#':
				if !used[geom.Pt(column, row)] {
					bodies = append(bodies, body{box: heroTiles(column, row, column+1, row+1), kind: '#'})
				}
			case '~':
				tileBox := heroTiles(column, row, column+1, row+1)
				if rough.IsZero() {
					rough = tileBox
					continue
				}
				rough = rough.Union(tileBox)
			case 'S', 'G':
				bodies = append(bodies, body{box: heroTiles(column, row, column+1, row+1), kind: tile})
			}
		}
	}

	return append(bodies, body{box: rough, kind: '~'})
}

func heroTiles(left, top, right, bottom int) geom.Box[float64] {
	return geom.Bx(geom.Pt(float64(left), float64(top)), geom.Pt(float64(right), float64(bottom)))
}

func heroCenter(bodies []body, kind byte) point {
	for _, b := range bodies {
		if b.kind == kind {
			return b.box.Center()
		}
	}

	panic(fmt.Sprintf("diagrams: no %q on the hero map", kind))
}

func heroPanel(t theme, caption string, area geom.Box[float64], meets func(geom.Box[float64]) bool) *panel {
	p := newPanel(t, caption, heroScale, heroCell)
	p.shade(area, meets)
	p.lattice(area)

	return p
}

func (p *panel) bodies(bodies []body, keys []int) {
	for key, b := range bodies {
		returned := slices.Contains(keys, key)
		switch b.kind {
		case '#':
			p.outline(b.box, style{fill: p.theme.wall})
		case '~':
			p.outline(b.box, style{fill: p.theme.leafSoft})
		case 'S':
			p.dot(b.box.Center(), p.theme.accent)
		case 'G':
			p.dot(b.box.Center(), p.theme.cool)
		default:
			panic(fmt.Sprintf("diagrams: unknown tile %q", b.kind))
		}
		if returned {
			p.outline(b.box, style{stroke: p.theme.accent, width: 2.5})
		}
	}
}
