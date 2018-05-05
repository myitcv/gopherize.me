// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package react

import (
	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

type BasicNode struct {
	o *js.Object
}

type BasicElement struct {
	*BasicNode
}

func newBasicElement() *BasicElement {
	return &BasicElement{
		BasicNode: &BasicNode{object.New()},
	}
}

type AriaSet map[string]string
type DataSet map[string]string

type BasicHTMLElement struct {
	*BasicElement

	AriaHasPopup   bool   `js:"aria-haspopup"`
	AriaExpanded   bool   `js:"aria-expanded"`
	AriaLabelledBy string `js:"aria-labelledby"`

	ID        string `js:"id" react:"omitempty"`
	Key       string `js:"key" react:"omitempty"`
	ClassName string `js:"className"`
	Role      string `js:"role"`
	Style     *CSS   `js:"style"`

	DataSet

	OnChange `js:"onChange"`
	OnClick  `js:"onClick"`

	Ref `js:"ref"`

	DangerouslySetInnerHTML *DangerousInnerHTML `js:"dangerouslySetInnerHTML"`
}

func newBasicHTMLElement() *BasicHTMLElement {
	return &BasicHTMLElement{
		BasicElement: newBasicElement(),
	}
}

type SyntheticEvent struct {
	o *js.Object

	PreventDefault  func() `js:"preventDefault"`
	StopPropagation func() `js:"stopPropagation"`
}

func (s *SyntheticEvent) Target() dom.HTMLElement {
	return dom.WrapHTMLElement(s.o.Get("target"))
}

type SyntheticMouseEvent struct {
	*SyntheticEvent

	ClientX int `js:"clientX"`
}
