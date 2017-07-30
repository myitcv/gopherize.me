package main

//go:generate reactGen

import (
	"math/rand"

	r "myitcv.io/react"
)

type OuterDef struct {
	r.ComponentDef
}

type OuterState struct {
	current *Gopher
	config  *Config
}

func Outer() *OuterElem {
	return buildOuterElem()
}

func (o OuterDef) ComponentWillMount() {
	o.SetState(OuterState{
		current: defaultGopher(hackConfig),
		config:  hackConfig,
	})
}

func (o OuterDef) Render() r.Element {
	debugln("Outer.Render")
	return r.Div(nil,
		Preview(PreviewProps{Current: o.State().current}),
		Chooser(ChooserProps{
			Current: o.State().current,
			Config:  o.State().config,
			Update:  o,
		}),
	)
}

func (o OuterDef) ResetGopher() {
	debugln("Outer.ResetGopher")
	s := o.State()
	s.current = defaultGopher(s.config)
	o.SetState(s)
}

func (o OuterDef) UpdateGopher(part int, val string) {
	debugln("Outer.UpdateGopher")
	s := o.State()

	nps := make([]string, len(s.current.Parts))
	copy(nps, s.current.Parts)
	nps[part] = val

	s.current = &Gopher{Parts: nps}
	o.SetState(s)
}

func (o OuterDef) RandomGopher() {
	debugln("Outer.RandomGopher")
	s := o.State()
	c := o.State().config

	var parts []string

	for _, cat := range c.Categories {
		p := cat.Options[rand.Intn(len(cat.Options))]
		parts = append(parts, p)
	}

	s.current = &Gopher{Parts: parts}
	o.SetState(s)
}

func randElem(ss []string) string {
	return ss[rand.Intn(len(ss))]
}

func defaultGopher(c *Config) *Gopher {
	parts := make([]string, len(c.Categories))

	parts[0] = c.Categories[0].Options[0]
	parts[1] = c.Categories[1].Options[0]

	return &Gopher{Parts: parts}
}
