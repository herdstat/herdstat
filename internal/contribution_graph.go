/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"
)

// ContributionRecord contains the activity data for a single day.
type ContributionRecord struct {
	Date  time.Time
	Count int
}

// ColorSpectrum defines a spectrum of colors given by two colors representing
// the left and right ends of the spectrum.
type ColorSpectrum struct {
	Min color.RGBA
	Max color.RGBA
}

// ColorScheme defines a color scheme for contribution graphs.
type ColorScheme struct {
	Light ColorSpectrum
	Dark  ColorSpectrum
}

// Coloring translates an intensity into a color. It is used to compute the
// color of graph cells and the legend.
type Coloring func(intensity uint8, darkScheme bool) color.RGBA

// GetColoring returns a linear coloring based on a single primary color.
func GetColoring(scheme ColorScheme) Coloring {
	return func(intensity uint8, darkScheme bool) color.RGBA {
		var spectrum ColorSpectrum
		if darkScheme {
			spectrum = scheme.Dark
		} else {
			spectrum = scheme.Light
		}
		return defaultColoring(spectrum.Min, spectrum.Max, intensity)
	}
}

func defaultColoring(min color.RGBA, max color.RGBA, intensity uint8) color.RGBA {
	m := func(a uint8, b uint8) uint8 {
		// TODO Get rid of float64?
		return a + uint8(math.Round(float64(b)-float64(a))/256.0*float64(intensity))
	}
	return color.RGBA{
		R: m(min.R, max.R),
		G: m(min.G, max.G),
		B: m(min.B, max.B),
	}
}

// ContributionGraph is a heatmap representation of 52 weeks of activity data.
type ContributionGraph struct {

	// 364 days (52 weeks) of activity data records
	Records []ContributionRecord

	// The date for the last day of activity.
	LastDate time.Time

	// Coloring defines the color of the graph cells.
	Coloring Coloring

	// The number of color levels
	Levels uint8
}

// NewContributionMap creates a new ContributionGraph.
func NewContributionMap(data []ContributionRecord, lastDate time.Time, coloring Coloring, levels uint8) *ContributionGraph {
	return &ContributionGraph{
		data,
		lastDate,
		coloring,
		levels,
	}
}

// intensity computes the intensity of the given ContributionRecord.
func (g *ContributionGraph) intensity(r ContributionRecord) uint8 {
	maxCount := max(g.Records, func(a, b ContributionRecord) int {
		return a.Count - b.Count
	}).Count
	if maxCount == 0 {
		return 0
	}
	return uint8(255.0 / maxCount * r.Count)
}

var (
	// The embedded stylesheet template used for styling the contribution graph.
	//go:embed contribution-graph.gohtml
	styleTemplate string
)

// StyleTemplateParams are the parameters used for rendering the stylesheet template.
type StyleTemplateParams struct {
	DarkColors  []color.RGBA
	LightColors []color.RGBA
}

// renderStyle writes the styleTemplate to the given decoder.
func (g *ContributionGraph) renderStyle(e *xml.Encoder) error {
	tmpl := template.Must(template.New("style").Parse(styleTemplate))
	var lightColors []color.RGBA
	for i := uint8(0); i < g.Levels; i++ {
		lightColors = append(lightColors, g.Coloring(uint8(uint(i)*255/(uint(g.Levels)-1)), false))
	}
	var darkColors []color.RGBA
	for i := uint8(0); i < g.Levels; i++ {
		darkColors = append(darkColors, g.Coloring(uint8(uint(i)*255/(uint(g.Levels)-1)), true))
	}
	params := StyleTemplateParams{
		DarkColors:  darkColors,
		LightColors: lightColors,
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, params); err != nil {
		return err
	}
	// Strip away the enclosing `style` tag (required to used via the encoder)
	styleTagStripped := strings.ReplaceAll(
		strings.ReplaceAll(buf.String(), "<style>", ""), "</style>", "")
	return style(e, styleTagStripped)
}

// Render writes the contribution map to the given xml.Encoder.
func (g *ContributionGraph) Render(e *xml.Encoder) error {

	// Write SVG opening tag
	err := e.EncodeToken(xml.StartElement{
		Name: xml.Name{
			Local: "svg",
		},
		Attr: []xml.Attr{
			{
				Name: xml.Name{
					Local: "xmlns",
				},
				Value: "http://www.w3.org/2000/svg",
			},
			cssClassAttr("herdstat-contribution-graph", "herdstat-contribution-graph-var"),
			{
				Name: xml.Name{
					Local: "width",
				},
				Value: "700",
			},
			{
				Name: xml.Name{
					Local: "height",
				},
				Value: "150",
			},
		},
	})
	if err != nil {
		return err
	}

	if err = g.renderStyle(e); err != nil {
		return err
	}

	if err = g.renderContributionCellMatrix(e); err != nil {
		return err
	}

	count := 0
	for _, record := range g.Records {
		count += record.Count
	}
	if err = g.renderOverallContributions(e, image.Point{
		X: 65,
		Y: 125,
	}, count); err != nil {
		return err
	}

	if err = g.renderLegend(e, image.Point{
		X: 565,
		Y: 125,
	}); err != nil {
		return err
	}

	// Write closing tag
	err = e.EncodeToken(xml.EndElement{
		Name: xml.Name{
			Local: "svg",
		},
	})
	if err != nil {
		return err
	}

	return err
}

func (g *ContributionGraph) renderContributionCellMatrix(e *xml.Encoder) error {
	if err := g.renderWeekdayAxis(e); err != nil {
		return err
	}

	// "Default" case of 51 full and 2 partial weeks
	location := image.Point{
		X: 50,
		Y: 10,
	}
	sliceCount := 53

	// Handle case of 52 full weeks, i.e., shift map one row to the right
	if g.LastDate.Weekday() == time.Saturday {
		location = location.Add(image.Point{X: 12})
		sliceCount = 52
	}
	err := translated(
		e,
		location,
		func(e *xml.Encoder) error {

			// Prepare the week slices
			var slices []weekSlice
			var sliceRecords []ContributionRecord
			records := make([]ContributionRecord, len(g.Records))
			copy(records, g.Records)
			for i := 0; i < sliceCount; i++ {
				var first = time.Sunday
				var last = time.Saturday
				switch i {
				case 0:
					first = (g.LastDate.Weekday() + 1) % 7
				case sliceCount - 1:
					last = g.LastDate.Weekday()
				}
				sliceRecords, records = take(records, int(last-first+1))
				ws, err := newWeekSlice(g, previousSunday(g.LastDate.AddDate(0, 0, -(sliceCount-i-1)*7)), first, last, sliceRecords, uint8(i))
				if err != nil {
					return err
				}
				slices = append(slices, *ws)
			}

			// Render heatmap
			for i, slice := range slices {
				err := translated(e, image.Point{X: 12 * i}, func(e *xml.Encoder) error {
					return slice.render(e, false)
				})
				if err != nil {
					return err
				}
			}

			// Render overlay
			for i, slice := range slices {
				err := translated(e, image.Point{X: 12 * i}, func(e *xml.Encoder) error {
					return slice.render(e, true)
				})
				if err != nil {
					return err
				}
			}

			return nil
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// renderWeekdayAxis renders the y-axis of the heatmap consisting of the days
// of the week.
func (g *ContributionGraph) renderWeekdayAxis(e *xml.Encoder) error {
	clsAttrs := cssClassAttrs("herdstat-contribution-graph-fg")
	err := simpleText(
		e,
		image.Point{
			X: 40,
			Y: 12 + 9 + 30,
		},
		end,
		clsAttrs,
		"Mon",
	)
	if err != nil {
		return err
	}

	err = simpleText(
		e,
		image.Point{
			X: 40,
			Y: 36 + 9 + 30,
		},
		end,
		clsAttrs,
		"Wed",
	)
	if err != nil {
		return err
	}

	err = simpleText(
		e,
		image.Point{
			X: 40,
			Y: 60 + 9 + 30,
		},
		end,
		clsAttrs,
		"Fri",
	)
	if err != nil {
		return err
	}

	return nil
}

// renderOverallContributions renders a label with the overall number of contributions.
func (g *ContributionGraph) renderOverallContributions(e *xml.Encoder, location image.Point, count int) error {
	return text(e, location.Add(image.Point{Y: 9}), start, cssClassAttrs("herdstat-contribution-graph-fg"),
		func(e *xml.Encoder) error {
			err := nonEmptyElement(e, xml.StartElement{
				Name: xml.Name{
					Local: "tspan",
				},
				Attr: []xml.Attr{
					{
						Name: xml.Name{
							Local: "font-weight",
						},
						Value: "800",
					},
				},
			}, func(e *xml.Encoder) error {
				return e.EncodeToken(xml.CharData(fmt.Sprintf("%d contributions\u00A0", count)))
			})
			if err != nil {
				return nil
			}
			return e.EncodeToken(xml.CharData("in the last year"))
		})
}

// renderLegend renders a legend for decoding contribution intensity
// indicators.
func (g *ContributionGraph) renderLegend(e *xml.Encoder, location image.Point) error {
	clsAttrs := cssClassAttrs("herdstat-contribution-graph-fg")
	err := simpleText(
		e,
		location.Add(image.Point{Y: 9}),
		start,
		clsAttrs,
		"Less",
	)
	if err != nil {
		return err
	}

	for i := 0; i < 5; i++ {
		level := (g.Levels - 1) / 4 * uint8(i)
		err := coloredRoundedRect(e, image.Point{
			X: location.X + 29 + i*12,
			Y: location.Y,
		}, cssClassAttrs(
			"herdstat-contribution-graph-cell",
			fmt.Sprintf("herdstat-contribution-graph-cell-L%d-bg", level)))
		if err != nil {
			return err
		}
	}

	err = simpleText(
		e,
		location.Add(image.Point{X: 29 + 5*12 + 1, Y: 9}),
		start,
		clsAttrs,
		"More",
	)
	if err != nil {
		return err
	}
	return nil
}

// weekSlice represents one (partial) week of activity data rendered as a
// vertical array of color-coded boxes.
type weekSlice struct {

	// TODO
	Graph *ContributionGraph

	// The date of the first day of the week (Sunday).
	Date time.Time

	// The first day of the potentially partial week.
	First time.Weekday

	// The last day of the potentially partial week.
	Last time.Weekday

	// The contribution records for this week. Must be of length
	// Last - First + 1.
	Records []ContributionRecord

	// TODO
	Index uint8
}

// newWeekSlice creates a new weekSlice. Arguments are checked for validity and
// errors are returned in case of violation.
func newWeekSlice(graph *ContributionGraph, refDate time.Time, first time.Weekday, last time.Weekday, records []ContributionRecord, index uint8) (*weekSlice, error) {
	if refDate.Weekday() != time.Sunday {
		return nil, errors.New("reference day must be a sunday")
	}
	if first != time.Sunday && last != time.Saturday {
		return nil, errors.New(fmt.Sprintf("either first must be %s or last must be %s", time.Sunday, time.Saturday))
	}
	expectedRecordCount := int(last - first + 1)
	if len(records) != expectedRecordCount {
		return nil, errors.New(fmt.Sprintf("wrong number of records, was %d but must be %d", len(records), expectedRecordCount))
	}
	return &weekSlice{
		graph, refDate, first, last, records, index,
	}, nil
}

// isFirstWeekOfMonth returns true iff the weekSlice represents the first week
// of a month.
func (w weekSlice) isFirstWeekOfMonth() bool {
	dayOfMonth := w.Date.Day()
	return dayOfMonth >= 1 && dayOfMonth <= 7
}

// render draws the weekSlice as a vertical array of color-coded boxes.
func (w weekSlice) render(e *xml.Encoder, overlay bool) error {
	if !overlay && w.isFirstWeekOfMonth() {
		ta := start
		dx := 0
		if w.Index == 52 {
			ta = end
			dx = 10
		}
		err := simpleText(e, image.Point{X: dx, Y: 10}, ta,
			cssClassAttrs("herdstat-contribution-graph-fg"), w.Date.Format("Jan"))
		if err != nil {
			return err
		}
	}
	return translated(e, image.Point{Y: 20}, func(e *xml.Encoder) error {
		for _, record := range w.Records {
			if err := w.renderDay(e, w.Index, record, overlay); err != nil {
				return err
			}
		}
		return nil
	})
}

// verticalPosition is used to specify the vertical position of an element.
type verticalPosition uint8

const (
	top verticalPosition = iota
	bottom
)

// horizontalPosition is used to specify the horizontal position of an element.
type horizontalPosition uint8

const (
	left horizontalPosition = iota
	center
	right
)

// position is used to specify the horizontal and vertical position of an
// element.
type position struct {
	horizontal horizontalPosition
	vertical   verticalPosition
}

// tooltipSize is the height and half-width of the tooltip "tip".
const tooltipSize = 5

// tooltipOffset is the vertical distance of the tooltip "tip" from the target.
const tooltipOffset = 10

// tooltipBoxOrigin computes the origin (upper left corner) of the rectangular
// box of the tooltip.
func (w weekSlice) tooltipBoxOrigin(location image.Point, tipPosition position, dimension image.Point) image.Point {
	var dx int
	switch tipPosition.horizontal {
	case left:
		dx = -4 * tooltipSize
	case center:
		dx = -dimension.X / 2
	case right:
		dx = -dimension.X + 4*tooltipSize
	}
	var dy int
	switch tipPosition.vertical {
	case top:
		dy = -(tooltipSize + dimension.Y + tooltipOffset)
	case bottom:
		dy = tooltipSize + tooltipOffset
	}
	return location.Add(image.Point{
		X: dx,
		Y: dy,
	})
}

// tooltipTrianglePoints returns a string encoded representation of the tooltip
// "tip" that can be used in an SVG polygon based on the given verticalPosition.
func (w weekSlice) tooltipTrianglePoints(location image.Point, position verticalPosition) string {
	var m int
	var offset int
	switch position {
	case top:
		m = tooltipSize
		offset = tooltipOffset
	case bottom:
		m = -tooltipSize
		offset = -tooltipOffset
	}
	return fmt.Sprintf(
		"%d,%d %d,%d %d,%d",
		location.X-tooltipSize, location.Y-m-offset,
		location.X+tooltipSize, location.Y-m-offset,
		location.X, location.Y-offset)
}

// renderTooltip renders a tooltip with activity information.
func (w weekSlice) renderTooltip(e *xml.Encoder, location image.Point, tipPosition position, record ContributionRecord) error {
	return nonEmptyElement(e, xml.StartElement{
		Name: xml.Name{Local: "g"},
		Attr: cssClassAttrs("herdstat-contribution-graph-cell-tooltip"),
	}, func(e *xml.Encoder) error {
		width := 230
		height := 30
		origin := w.tooltipBoxOrigin(location, tipPosition, image.Point{
			X: width,
			Y: height,
		})
		err := emptyElement(e, xml.StartElement{
			Name: xml.Name{
				Local: "rect",
			},
			Attr: []xml.Attr{
				{
					Name: xml.Name{
						Local: "x",
					},
					Value: strconv.Itoa(origin.X),
				},
				{
					Name: xml.Name{
						Local: "y",
					},
					Value: strconv.Itoa(origin.Y),
				},
				{
					Name: xml.Name{
						Local: "width",
					},
					Value: strconv.Itoa(width),
				},
				{
					Name: xml.Name{
						Local: "height",
					},
					Value: strconv.Itoa(height),
				},
				{
					Name: xml.Name{
						Local: "rx",
					},
					Value: "4",
				},
			},
		})
		if err != nil {
			return err
		}

		err = emptyElement(e, xml.StartElement{
			Name: xml.Name{Local: "polygon"},
			Attr: []xml.Attr{
				{
					Name: xml.Name{
						Local: "points",
					},
					Value: w.tooltipTrianglePoints(location, tipPosition.vertical),
				},
			},
		})
		if err != nil {
			return err
		}

		return text(e,
			image.Point{
				X: origin.X + width/2,
				Y: origin.Y + height/2 + 4,
			},
			middle,
			[]xml.Attr{},
			func(e *xml.Encoder) error {
				err := nonEmptyElement(e, xml.StartElement{
					Name: xml.Name{
						Local: "tspan",
					},
					Attr: []xml.Attr{
						{
							Name: xml.Name{
								Local: "font-weight",
							},
							Value: "800",
						},
					},
				}, func(e *xml.Encoder) error {
					return e.EncodeToken(xml.CharData(fmt.Sprintf("%d contributions\u00A0", record.Count)))
				})
				if err != nil {
					return nil
				}
				return e.EncodeToken(xml.CharData(fmt.Sprintf("on %s", record.Date.Format("Jan 2, 2006"))))
			},
		)
	})
}

// renderDay draws a single color-coded box representing a single day of
// contributions.
func (w weekSlice) renderDay(e *xml.Encoder, weekIndex uint8, record ContributionRecord, overlay bool) error {
	y := int(record.Date.Weekday()) * 12
	col := uint8(math.Min(math.Ceil(float64(w.Graph.intensity(record))/256.0*float64(w.Graph.Levels)), float64(w.Graph.Levels-1)))
	var attrs []xml.Attr
	if overlay {
		attrs = []xml.Attr{
			{
				Name: xml.Name{
					Local: "fill-opacity",
				},
				Value: "0.0",
			},
			cssClassAttr("herdstat-contribution-graph-cell-overlay"),
		}
	} else {
		attrs = cssClassAttrs(
			"herdstat-contribution-graph-cell",
			fmt.Sprintf("herdstat-contribution-graph-cell-L%d-bg", col))
	}
	err := coloredRoundedRect(e, image.Point{
		X: 0,
		Y: y,
	}, attrs)
	if err != nil {
		return err
	}
	var xpos horizontalPosition
	switch {
	case weekIndex < 10:
		xpos = left
	case weekIndex > 42:
		xpos = right
	default:
		xpos = center
	}
	var vpos verticalPosition
	switch {
	case record.Date.Weekday() <= 2:
		vpos = bottom
	default:
		vpos = top
	}
	if overlay {
		err = w.renderTooltip(e, image.Point{
			X: 5,
			Y: y + 5,
		}, position{
			horizontal: xpos,
			vertical:   vpos,
		}, record)
		return err
	}
	return nil
}
