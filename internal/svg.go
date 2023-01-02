/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"strconv"
)

// toHex converts the given RGBA color into its RGB hex representation.
func toHex(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// contentProducer emits XML content by using the given xml.Encoder.
type contentProducer func(e *xml.Encoder) error

// nonEmptyElement writes the given element as a non-empty XML element wrapping
// the content produced by the given contentProducer using the given
// xml.Encoder.
func nonEmptyElement(e *xml.Encoder, element xml.StartElement, content contentProducer) error {
	if err := e.EncodeToken(element); err != nil {
		return err
	}
	if err := content(e); err != nil {
		return err
	}
	return e.EncodeToken(xml.EndElement{
		Name: element.Name,
	})
}

// emptyElement writes the given element as an empty XML element using the
// given xml.Encoder.
func emptyElement(e *xml.Encoder, element xml.StartElement) error {
	return nonEmptyElement(e, element, func(e *xml.Encoder) error {
		return nil
	})
}

// translated translates the body by the given offset.
func translated(e *xml.Encoder, offset image.Point, content contentProducer) error {
	return nonEmptyElement(e, xml.StartElement{
		Name: xml.Name{
			Local: "g",
		},
		Attr: []xml.Attr{
			{
				Name:  xml.Name{Local: "transform"},
				Value: fmt.Sprintf("translate(%d %d)", offset.X, offset.Y),
			},
		},
	}, content)
}

// textAnchor is used to align (start-, middle- or end-alignment) a string of
// pre-formatted text. For more details see [mdn].
//
// [mdn]: https://developer.mozilla.org/en-US/docs/Web/SVG/Attribute/text-anchor
type textAnchor uint8

const (

	// start aligns text such that the start of the text string is at the
	// initial current text position.
	start textAnchor = iota

	// middle aligns text such that the middle of the text string is at the
	// current text position.
	middle

	// end aligns text such that the end of the resulting rendered text is at
	// the initial current text position.
	end
)

// String generates a text representation of a textAnchor.
func (t textAnchor) String() string {
	switch t {
	case start:
		return "start"
	case middle:
		return "middle"
	case end:
		return "end"
	}
	panic("unknown text anchor constant")
}

// simpleText renders text at the given position using the given textAnchor.
func simpleText(e *xml.Encoder, location image.Point, fill color.RGBA, anchor textAnchor, content string) error {
	return text(e, location, fill, anchor, func(e *xml.Encoder) error {
		return e.EncodeToken(xml.CharData(content))
	})
}

// text renders complex text (e.g., that includes tspan elements) at the given
// position using the given textAnchor.
func text(e *xml.Encoder, location image.Point, fill color.RGBA, anchor textAnchor, content contentProducer) error {
	return nonEmptyElement(e, xml.StartElement{
		Name: xml.Name{
			Local: "text",
		},
		Attr: []xml.Attr{
			{
				Name: xml.Name{
					Local: "x",
				},
				Value: strconv.Itoa(location.X),
			},
			{
				Name: xml.Name{
					Local: "y",
				},
				Value: strconv.Itoa(location.Y),
			},
			{
				Name: xml.Name{
					Local: "font-size",
				},
				Value: "12px",
			},
			{
				Name: xml.Name{
					Local: "fill",
				},
				Value: toHex(fill),
			},
			{
				Name: xml.Name{
					Local: "text-anchor",
				},
				Value: anchor.String(),
			},
		},
	}, content)
}

// coloredRoundedRect renders a filled rectangle at the given location.
func coloredRoundedRect(e *xml.Encoder, location image.Point, color color.RGBA, attrs []xml.Attr) error {
	allAttrs := []xml.Attr{
		{
			Name: xml.Name{
				Local: "x",
			},
			Value: strconv.Itoa(location.X),
		},
		{
			Name: xml.Name{
				Local: "y",
			},
			Value: strconv.Itoa(location.Y),
		},
		{
			Name: xml.Name{
				Local: "width",
			},
			Value: strconv.Itoa(10),
		},
		{
			Name: xml.Name{
				Local: "height",
			},
			Value: strconv.Itoa(10),
		},
		{
			Name: xml.Name{
				Local: "rx",
			},
			Value: strconv.Itoa(2),
		},
		{
			Name: xml.Name{
				Local: "fill",
			},
			Value: toHex(color),
		},
	}
	for _, attr := range attrs {
		allAttrs = append(allAttrs, attr)
	}
	return emptyElement(e, xml.StartElement{
		Name: xml.Name{
			Local: "rect",
		},
		Attr: allAttrs,
	})
}
