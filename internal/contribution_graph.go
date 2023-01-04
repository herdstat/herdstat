/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"time"
)

// legendTextColor is used for rendering text used in legends.
var legendTextColor = color.RGBA{
	R: 176,
	G: 186,
	B: 199,
}

// tooltipTextColor is used for rendering text in tooltips.
var tooltipTextColor = color.RGBA{
	R: 207,
	G: 217,
	B: 228,
}

// ContributionRecord contains the activity data for a single day.
type ContributionRecord struct {
	Date  time.Time
	Count int
}

// Coloring translates an intensity into a color. It is used to compute the
// color of graph cells and the legend.
type Coloring func(intensity uint8) color.RGBA

// GetColoring returns a linear coloring based on a single color.
func GetColoring(highest color.RGBA) Coloring {
	return func(intensity uint8) color.RGBA {
		return defaultColoring(highest, intensity)
	}
}

// GitHubColoring uses the colors used by GitHub contribution charts.
var GitHubColoring = GetColoring(color.RGBA{
	R: 57,
	G: 211,
	B: 82,
})

func defaultColoring(highest color.RGBA, intensity uint8) color.RGBA {
	m := func(a uint8, b uint8) uint8 {
		// TODO Get rid of float64?
		return a + uint8(math.Round(float64(b)-float64(a))/256.0*float64(intensity))
	}
	from := color.RGBA{
		R: 45,
		G: 51,
		B: 59,
	}
	return color.RGBA{
		R: m(from.R, highest.R),
		G: m(from.G, highest.G),
		B: m(from.B, highest.B),
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
}

// NewContributionMap creates a new ContributionGraph.
func NewContributionMap(data []ContributionRecord, lastDate time.Time) *ContributionGraph {
	return &ContributionGraph{
		data,
		lastDate,
		GitHubColoring,
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

	err = nonEmptyElement(e, xml.StartElement{
		Name: xml.Name{
			Local: "style",
		},
		Attr: []xml.Attr{
			{
				Name: xml.Name{
					Local: "type",
				},
				Value: "text/css",
			},
		},
	}, func(e *xml.Encoder) error {
		style := `
svg {
	font-family: -apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans",Helvetica,Arial,sans-serif;
} 

.tooltip { 
	visibility: hidden; 
	transition: opacity 0.3s;
} 

.day:hover + .tooltip {
	visibility: visible;
}`
		return e.EncodeToken(xml.CharData(style))
	})

	if err = g.renderWeekdayAxis(e); err != nil {
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
	err = translated(
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
				err = translated(e, image.Point{X: 12 * i}, func(e *xml.Encoder) error {
					return slice.render(e, false)
				})
				if err != nil {
					return err
				}
			}

			// Render overlay
			for i, slice := range slices {
				err = translated(e, image.Point{X: 12 * i}, func(e *xml.Encoder) error {
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

	err = g.renderLegend(e, image.Point{
		X: 565,
		Y: 125,
	})
	if err != nil {
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

// renderWeekdayAxis renders the y-axis of the heatmap consisting of the days
// of the week.
func (g *ContributionGraph) renderWeekdayAxis(e *xml.Encoder) error {

	err := simpleText(
		e,
		image.Point{
			X: 40,
			Y: 12 + 9 + 30,
		},
		legendTextColor,
		end,
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
		legendTextColor,
		end,
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
		legendTextColor,
		end,
		"Fri",
	)
	if err != nil {
		return err
	}

	return nil
}

// renderLegend renders a legend for decoding contribution intensity
// indicators.
func (g *ContributionGraph) renderLegend(e *xml.Encoder, location image.Point) error {
	err := simpleText(
		e,
		location.Add(image.Point{Y: 9}),
		legendTextColor,
		start,
		"Less",
	)
	if err != nil {
		return err
	}

	for i := 0; i < 5; i++ {
		err := coloredRoundedRect(e, image.Point{
			X: location.X + 29 + i*12,
			Y: location.Y,
		}, g.Coloring(uint8(255/4*i)), []xml.Attr{
			{ // TODO Duplicate
				Name: xml.Name{
					Local: "stroke",
				},
				Value: "white",
			},
			{
				Name: xml.Name{
					Local: "stroke-opacity",
				},
				Value: "0.05",
			},
		})
		if err != nil {
			return err
		}
	}

	err = simpleText(
		e,
		location.Add(image.Point{X: 29 + 5*12 + 1, Y: 9}),
		legendTextColor,
		start,
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
		err := simpleText(e, image.Point{X: dx, Y: 10}, legendTextColor, ta, w.Date.Format("Jan"))
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
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "class"}, Value: "tooltip"},
		},
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
				{
					Name: xml.Name{
						Local: "fill",
					},
					Value: "#656E7A",
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
				{
					Name: xml.Name{
						Local: "fill",
					},
					Value: "#656E7A",
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
			tooltipTextColor,
			middle,
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
					return e.EncodeToken(xml.CharData(fmt.Sprintf("%d contributions", record.Count)))
				})
				if err != nil {
					return nil
				}
				return e.EncodeToken(xml.CharData(fmt.Sprintf(" on %s", record.Date.Format("Jan 2, 2006"))))
			},
		)
	})
}

// renderDay draws a single color-coded box representing a single day of
// contributions.
func (w weekSlice) renderDay(e *xml.Encoder, weekIndex uint8, record ContributionRecord, overlay bool) error {
	y := int(record.Date.Weekday()) * 12
	col := w.Graph.Coloring(w.Graph.intensity(record))
	var attrs []xml.Attr
	if overlay {
		attrs = []xml.Attr{
			{
				Name: xml.Name{
					Local: "fill-opacity",
				},
				Value: "0.0",
			},
			{
				Name: xml.Name{
					Local: "class",
				},
				Value: "day",
			},
		}
	} else {
		attrs = []xml.Attr{
			{
				Name: xml.Name{
					Local: "stroke",
				},
				Value: "white",
			},
			{
				Name: xml.Name{
					Local: "stroke-opacity",
				},
				Value: "0.05",
			},
		}
	}
	err := coloredRoundedRect(e, image.Point{
		X: 0,
		Y: y,
	}, col, attrs)
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
