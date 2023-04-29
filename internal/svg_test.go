/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"image"
)

var (
	emptyProducer = func(e *xml.Encoder) error {
		return nil
	}
	elem     = "elem"
	offset   = image.Point{X: 100, Y: 0}
	location = image.Point{X: 0, Y: 100}
	anchor   = start
	attrs    = []xml.Attr{{Name: xml.Name{Local: "name"}, Value: "value"}}
	content  = "text"
)

var _ = Describe("Generating a class attribute", func() {
	When("no class names given", func() {
		It("has the right attribute name with an empty value", func() {
			attr := cssClassAttr()
			Expect(attr.Name.Local).To(Equal("class"))
			Expect(attr.Value).To(BeEmpty())
		})
	})
	When("for two class names", func() {
		It("has the right attribute name and the value contains both classes", func() {
			classes := []string{"cls-a", "cls-b"}
			attr := cssClassAttr(classes[0], classes[1])
			Expect(attr.Name.Local).To(Equal("class"))
			Expect(attr.Value).To(Equal(fmt.Sprintf("%s %s", classes[0], classes[1])))
		})
	})
})

var _ = Describe("Generating an array of class attributes", func() {
	When("for two class names", func() {
		It("has the right attribute name and the value contains both classes", func() {
			classes := []string{"cls-a", "cls-b"}
			attrs := cssClassAttrs(classes[0], classes[1])
			Expect(attrs).To(HaveLen(1))
			attr := attrs[0]
			Expect(attr.Name.Local).To(Equal("class"))
			Expect(attr.Value).To(Equal(fmt.Sprintf("%s %s", classes[0], classes[1])))
		})
	})
})

var _ = Describe("Generating a non-empty element", func() {
	When("given a content provider that produces an error", func() {
		It("returns the produced error", func() {
			var buf bytes.Buffer
			enc := xml.NewEncoder(&buf)
			producedErr := errors.New("error")
			producer := func(e *xml.Encoder) error {
				return producedErr
			}
			err := nonEmptyElement(enc, xml.StartElement{Name: xml.Name{Local: elem}}, producer)
			Expect(err).To(Equal(producedErr))
		})
	})
	When("given an 'empty' content provider", func() {
		It("returns just the start and end element", func() {
			var buf bytes.Buffer
			enc := xml.NewEncoder(&buf)
			err := nonEmptyElement(enc, xml.StartElement{Name: xml.Name{Local: elem}}, emptyProducer)
			Expect(err).NotTo(HaveOccurred())
			_ = enc.Flush()
			Expect(buf.String()).To(Equal(fmt.Sprintf("<%s></%s>", elem, elem)))
		})
	})
})

var _ = Describe("Generating an empty element", func() {
	It("returns just the start and end element", func() {
		var buf bytes.Buffer
		enc := xml.NewEncoder(&buf)
		err := emptyElement(enc, xml.StartElement{Name: xml.Name{Local: elem}})
		Expect(err).NotTo(HaveOccurred())
		_ = enc.Flush()
		Expect(buf.String()).To(Equal(fmt.Sprintf("<%s></%s>", elem, elem)))
	})
})

var _ = Describe("Translating content", func() {
	When("given a non-zero offset", func() {
		It("wraps the given content into a group with the respective 'transform' attribute", func() {
			var buf bytes.Buffer
			enc := xml.NewEncoder(&buf)
			err := translated(enc, offset, emptyProducer)
			Expect(err).NotTo(HaveOccurred())
			_ = enc.Flush()
			Expect(buf.String()).To(Equal(fmt.Sprintf("<g transform=\"translate(%d %d)\"></g>", offset.X, offset.Y)))
		})
	})
})

var _ = Describe("Stringify text anchor", func() {
	When("given a valid textAnchor", func() {
		It("yields the correct string", func() {
			Expect(start.String()).To(Equal("start"))
			Expect(middle.String()).To(Equal("middle"))
			Expect(end.String()).To(Equal("end"))
		})
	})
	When("given an invalid textAnchor", func() {
		It("panics", func() {
			Expect(func() { _ = textAnchor(3).String() }).To(Panic())
		})
	})
})

var _ = Describe("Rendering text", func() {
	Context("given all attributes and an empty producer", func() {
		When("using the generic text function", func() {
			It("outputs an empty 'text' element with all expected attributes", func() {
				var buf bytes.Buffer
				enc := xml.NewEncoder(&buf)
				producer := func(e *xml.Encoder) error {
					return e.EncodeToken(xml.CharData(content))
				}
				err := text(enc, location, start, attrs, producer)
				Expect(err).NotTo(HaveOccurred())
				_ = enc.Flush()
				Expect(buf.String()).To(Equal(
					fmt.Sprintf("<text x=\"%d\" y=\"%d\" font-size=\"12px\" text-anchor=\"%s\" %s=\"%s\">%s</text>",
						location.X, location.Y, anchor, attrs[0].Name.Local, attrs[0].Value, content)))
			})
		})
		When("using the simple text function", func() {
			It("outputs an empty 'text' element with all expected attributes", func() {
				var buf bytes.Buffer
				enc := xml.NewEncoder(&buf)
				err := simpleText(enc, location, start, attrs, content)
				Expect(err).NotTo(HaveOccurred())
				_ = enc.Flush()
				Expect(buf.String()).To(Equal(
					fmt.Sprintf("<text x=\"%d\" y=\"%d\" font-size=\"12px\" text-anchor=\"%s\" %s=\"%s\">%s</text>",
						location.X, location.Y, anchor, attrs[0].Name.Local, attrs[0].Value, content)))
			})
		})
	})
})

var _ = When("Generating a colored round rect", func() {
	It("returns a rect element with the correct position and the given attributes", func() {
		var buf bytes.Buffer
		enc := xml.NewEncoder(&buf)
		err := coloredRoundedRect(enc, location, attrs)
		Expect(err).NotTo(HaveOccurred())
		_ = enc.Flush()
		Expect(buf.String()).To(Equal(fmt.Sprintf("<rect x=\"%d\" y=\"%d\" rx=\"2\" %s=\"%s\"></rect>",
			location.X, location.Y, attrs[0].Name.Local, attrs[0].Value)))
	})
})

var _ = When("Generating a style element", func() {
	It("returns a style element with the right type and the given directives", func() {
		directives := "test"
		var buf bytes.Buffer
		enc := xml.NewEncoder(&buf)
		err := style(enc, directives)
		Expect(err).NotTo(HaveOccurred())
		_ = enc.Flush()
		Expect(buf.String()).To(Equal(fmt.Sprintf("<style type=\"text/css\">%s</style>", directives)))
	})
})
