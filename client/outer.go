package main

//go:generate reactGen

import (
	r "myitcv.io/react"
)

type Gopher struct {
	Parts []string
}

type Config struct {
	Categories []Category
}

type Category struct {
	Name    string
	Options []string
}

type OuterDef struct {
	r.ComponentDef
}

type OuterState struct {
	current *Gopher
	config  *Config
}

func Outer() *OuterDef {
	res := new(OuterDef)
	r.BlessElement(res, nil)
	return res
}

func (o *OuterDef) ComponentWillMount() {
	o.SetState(OuterState{
		current: defaultGopher(hackConfig),
		config:  hackConfig,
	})
}

func (o *OuterDef) Render() r.Element {
	return r.Div(nil,
		Preview(PreviewProps{Current: o.State().current}),
		Chooser(ChooserProps{
			Config:  o.State().config,
			Current: o.State().current,
			Update:  o,
		}),
	)
}

func (o *OuterDef) UpdateGopher(g *Gopher) {
	s := o.State()

	if g == nil {
		g = defaultGopher(s.config)
	}

	s.current = g
	o.SetState(s)
}

func defaultGopher(c *Config) *Gopher {
	parts := make([]string, len(c.Categories))

	parts[0] = c.Categories[0].Options[0]
	parts[1] = c.Categories[1].Options[0]

	return &Gopher{Parts: parts}
}

type UpdateGopher interface {
	UpdateGopher(g *Gopher)
}
