package main

import (
	"slices"

	geom "github.com/gravitton/geometry"
	"github.com/gravitton/spatial"
)

const cellSize = 32.0

var yard = geom.Bx(geom.Pt(0.0, 0.0), geom.Pt(192.0, 96.0))

var entities = []geom.Box[float64]{
	geom.BoxFromMin(geom.Pt(6.0, 8.0), geom.Sz(20.0, 16.0)),
	geom.BoxFromMin(geom.Pt(38.0, 10.0), geom.Sz(24.0, 12.0)),
	geom.BoxFromMin(geom.Pt(52.0, 30.0), geom.Sz(14.0, 30.0)),
	geom.BoxFromMin(geom.Pt(76.0, 6.0), geom.Sz(28.0, 18.0)),
	geom.BoxFromMin(geom.Pt(114.0, 14.0), geom.Sz(12.0, 12.0)),
	geom.BoxFromMin(geom.Pt(140.0, 8.0), geom.Sz(30.0, 22.0)),
	geom.BoxFromMin(geom.Pt(8.0, 44.0), geom.Sz(16.0, 24.0)),
	geom.BoxFromMin(geom.Pt(30.0, 70.0), geom.Sz(26.0, 18.0)),
	geom.BoxFromMin(geom.Pt(80.0, 48.0), geom.Sz(32.0, 24.0)),
	geom.BoxFromMin(geom.Pt(98.0, 76.0), geom.Sz(12.0, 14.0)),
	geom.BoxFromMin(geom.Pt(130.0, 50.0), geom.Sz(22.0, 28.0)),
	geom.BoxFromMin(geom.Pt(162.0, 56.0), geom.Sz(22.0, 16.0)),
	geom.BoxFromMin(geom.Pt(150.0, 76.0), geom.Sz(10.0, 12.0)),
	geom.BoxFromMin(geom.Pt(176.0, 8.0), geom.Sz(10.0, 20.0)),
}

func indexOf(boxes []geom.Box[float64], size float64) *spatial.Hash[int, float64] {
	index := spatial.NewHash[int](size)
	for key, box := range boxes {
		index.Set(key, box)
	}

	return index
}

func yardPanel(t theme, caption string, scale float64, meets func(geom.Box[float64]) bool) *panel {
	p := newPanel(t, caption, scale, cellSize)
	p.shade(yard, meets)
	p.lattice(yard)

	return p
}

func (p *panel) stored(keys []int) {
	for key, box := range entities {
		if slices.Contains(keys, key) {
			p.outline(box, style{fill: p.theme.accentSoft, stroke: p.theme.accent, width: 2})
			continue
		}

		p.outline(box, style{stroke: p.theme.muted, width: 1.5})
	}
}
