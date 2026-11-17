package main

import (
	"fmt"
	"math"

	geom "github.com/gravitton/geometry"
)

var figures = []figure{
	{"hero", hero},
	{"queries", queries},
	{"casts", casts},
	{"once", once},
}

func queries(t theme) []*panel {
	index := indexOf(entities, cellSize)
	camera := geom.BoxFromMin(geom.Pt(44.0, 18.0), geom.Sz(68.0, 50.0))
	cursor := geom.Pt(92.0, 60.0)

	inBox := yardPanel(t, "QueryBox(camera)", 2.25, camera.IntersectsBox)
	boxKeys := index.QueryBox(camera)
	inBox.stored(boxKeys)
	inBox.outline(camera, style{stroke: t.cool, width: 2.5, dashed: true})
	inBox.note = fmt.Sprintf("%d keys", len(boxKeys))

	atPoint := yardPanel(t, "QueryPoint(cursor)", 2.25, func(box geom.Box[float64]) bool {
		return box.Contains(cursor)
	})
	pointKeys := index.QueryPoint(cursor)
	atPoint.stored(pointKeys)
	atPoint.dot(cursor, t.cool)
	atPoint.note = fmt.Sprintf("%d key", len(pointKeys))

	return []*panel{inBox, atPoint}
}

func casts(t theme) []*panel {
	index := indexOf(entities, cellSize)
	guard, target := geom.Pt(4.0, 90.0), geom.Pt(188.0, 40.0)
	sight := geom.Seg(guard, target)

	p := yardPanel(t, "QuerySegment(sight)", 4, sight.IntersectsBox)
	var order []int
	for key := range index.QuerySegment(sight) {
		order = append(order, key)
	}
	p.stored(order)
	p.segment(sight, t.cool, false)
	p.dot(guard, t.cool)
	p.dot(target, t.cool)
	p.label(guard.AddXY(10, -8), "guard", t.cool)
	p.label(target.AddXY(-12, -8), "target", t.cool)
	for n, key := range order {
		p.badge(entities[key].Center(), fmt.Sprint(n+1))
	}
	p.note = "keys in the order QuerySegment yields them; a loop that breaks reads only the first"

	return []*panel{p}
}

func once(t theme) []*panel {
	index := indexOf(entities, cellSize)
	key := 8
	query := geom.BoxFromMin(geom.Pt(84.0, 40.0), geom.Sz(40.0, 34.0))
	if keys := index.QueryBox(query); len(keys) != 1 || keys[0] != key {
		panic(fmt.Sprintf("diagrams: QueryBox(%s) returned %v, not key %d once", query, keys, key))
	}

	box := entities[key]
	shared := func(cell geom.Box[float64]) bool {
		return cell.IntersectsBox(box) && cell.IntersectsBox(query)
	}
	first := geom.Pt(
		math.Max(math.Floor(box.Min.X/cellSize), math.Floor(query.Min.X/cellSize)),
		math.Max(math.Floor(box.Min.Y/cellSize), math.Floor(query.Min.Y/cellSize)),
	).Multiply(cellSize)

	view := geom.Bx(geom.Pt(32.0, 32.0), geom.Pt(160.0, 96.0))
	p := yardPanel(t, "QueryBox(query)", 4, shared)
	p.crop(view)
	for k, other := range entities {
		if k != key {
			p.outline(other, style{stroke: t.muted, width: 1.5})
		}
	}
	p.outline(box, style{fill: t.accentSoft, stroke: t.accent, width: 2})
	p.outline(query, style{stroke: t.cool, width: 2.5, dashed: true})
	for _, cell := range p.cells(view) {
		if !shared(cell) {
			continue
		}

		quarter := overlapOf(overlapOf(cell, box), query)

		if cell.Min == first {
			p.tag(quarter.Center(), "reported", t.accent)
			continue
		}

		p.tag(quarter.Center(), "skipped", t.muted)
	}
	p.note = "the key is in four of the cells the query covers, and is reported in the first"

	return []*panel{p}
}

func overlapOf(a, b geom.Box[float64]) geom.Box[float64] {
	return geom.Bx(
		geom.Pt(math.Max(a.Min.X, b.Min.X), math.Max(a.Min.Y, b.Min.Y)),
		geom.Pt(math.Min(a.Max.X, b.Max.X), math.Min(a.Max.Y, b.Max.Y)),
	)
}
