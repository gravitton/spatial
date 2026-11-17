package main

import (
	"math"

	geom "github.com/gravitton/geometry"
)

type panel struct {
	canvas
	scale float64
	size  float64
}

func newPanel(t theme, caption string, scale, size float64) *panel {
	return &panel{canvas: newCanvas(t, caption), scale: scale, size: size}
}

func (p *panel) crop(area geom.Box[float64]) {
	p.clip(geom.Bx(p.pixel(area.Min), p.pixel(area.Max)))
}

func (p *panel) shade(area geom.Box[float64], meets func(geom.Box[float64]) bool) {
	for _, box := range p.cells(area) {
		fill := p.theme.cell
		if meets != nil && meets(box) {
			fill = p.theme.shade
		}

		p.area(box.Polygon().Points, style{fill: fill})
	}
}

func (p *panel) lattice(area geom.Box[float64]) {
	for _, box := range p.cells(area) {
		p.area(box.Polygon().Points, style{stroke: p.theme.edge, width: 1})
	}
}

func (p *panel) cells(area geom.Box[float64]) []geom.Box[float64] {
	first := geom.Pt(math.Floor(area.Min.X/p.size), math.Floor(area.Min.Y/p.size))
	last := geom.Pt(math.Ceil(area.Max.X/p.size), math.Ceil(area.Max.Y/p.size))

	var boxes []geom.Box[float64]
	for y := first.Y; y < last.Y; y++ {
		for x := first.X; x < last.X; x++ {
			boxes = append(boxes, geom.BoxFromMin(geom.Pt(x*p.size, y*p.size), geom.SzU(p.size)))
		}
	}

	return boxes
}

func (p *panel) area(points []point, s style) {
	p.polygon(p.pixels(points), s)
}

func (p *panel) outline(box geom.Box[float64], s style) {
	p.area(box.Polygon().Points, s)
}

func (p *panel) segment(segment geom.Segment[float64], color string, dashed bool) {
	p.line(p.pixel(segment.Start), p.pixel(segment.End), style{stroke: color, width: 2, dashed: dashed})
}

func (p *panel) dot(at point, color string) {
	p.circle(p.pixel(at), 4, style{fill: color})
}

func (p *panel) label(at point, text, color string) {
	p.text(p.pixel(at), font{size: 11, halo: true}, span{text, color})
}

func (p *panel) badge(at point, text string) {
	p.circle(p.pixel(at), 10, style{fill: p.theme.accent})
	p.text(p.pixel(at), font{size: 12}, span{text, p.theme.onAccent})
}

func (p *panel) tag(at point, text, color string) {
	p.text(p.pixel(at), font{size: 8}, span{text, color})
}

func (p *panel) pixel(at point) point {
	return at.Multiply(p.scale)
}

func (p *panel) pixels(points []point) []point {
	result := make([]point, len(points))
	for i, at := range points {
		result[i] = p.pixel(at)
	}

	return result
}
