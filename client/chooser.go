package main

//go:generate reactGen

import (
	"math/rand"
	"path/filepath"

	r "myitcv.io/react"
	"myitcv.io/react/jsx"
)

type ChooserProps struct {
	Current *Gopher
	Config  *Config
	Update  UpdateGopher
}

type ChooserState struct {
	open int
}

type ChooserDef struct {
	r.ComponentDef
}

func Chooser(p ChooserProps) *ChooserDef {
	res := new(ChooserDef)
	r.BlessElement(res, p)
	return res
}

func (ch *ChooserDef) Render() r.Element {
	var catDivs []r.Element

	for i, cat := range ch.Props().Config.Categories {
		catDivs = append(catDivs, ch.buildPanel(cat, i))
	}

	args := []r.Element{
		r.Button(
			&r.ButtonProps{
				ID:        "shuffle-button",
				ClassName: "btn btn-default",
				OnClick:   shuffleClick{ch},
			},
			r.I(&r.IProps{ClassName: "glyphicon glyphicon-refresh"}),
			r.S("Shuffle"),
		),
		r.Button(
			&r.ButtonProps{
				ID:        "reset-button",
				ClassName: "btn btn-default",
				OnClick:   resetClick{ch},
			},
			r.S("Reset"),
		),
		r.BR(nil),
		r.BR(nil),
		r.Div(
			&r.DivProps{
				ClassName: "panel-group",
				ID:        "options",
				Role:      "tablist",
			},
			catDivs...,
		),
		jsx.HTMLElem(`
			<div>
				<div classname="panel panel-default">
					<div classname="panel-body text-right" style="overflow-y: hidden">
						<button id='next-button' classname='btn btn-primary btn-lg'>
							Save &amp; continue&hellip;
							<i classname='glyphicon glyphicon glyphicon-chevron-right'></i>
						</button>
					</div>
				</div>
				<footer>
					Be truly unique, there are
					<span classname='total_combinations'></span>
					<hr/>
					Artwork by <a href='https://twitter.com/ashleymcnamara' target='_blank'>Ashley McNamara</a><br />inspired by <a href='http://reneefrench.blogspot.co.uk/' target='_blank'>Renee French</a><br />
					Web app by <a href='https://twitter.com/matryer' target='_blank'>Mat Ryer</a>
					<hr>
					<a href='https://github.com/matryer/gopherize.me'>View on GitHub</a>
					●
					<a href='/branding'>Add your brand</a>
				</footer>
			</div>
		`),
	}

	return r.Div(&r.DivProps{ClassName: "col-xs-4"}, args...)
}

func (ch *ChooserDef) buildPanel(c Category, i int) *r.DivDef {
	collapse := " collapse"

	if i == ch.State().open {
		collapse = ""
	}

	var imgs []r.Element

	for _, o := range c.Options {
		imgs = append(imgs,
			r.Label(
				&r.LabelProps{ClassName: "item"},
				r.Img(
					&r.ImgProps{Src: filepath.Join("..", "artwork", o+"_thumbnail.png")},
				),
			),
		)
	}

	return r.Div(&r.DivProps{ClassName: "panel panel-default"},
		r.Div(&r.DivProps{ClassName: "panel-heading", Role: "tab"},
			r.H4(
				&r.H4Props{ClassName: "panel-title"},
				r.A(
					&r.AProps{
						OnClick: expandClick{ch: ch, i: i},
					},
					r.S(c.Name),
				),
			),
		),
		r.Div(
			&r.DivProps{
				ID:        "Body",
				ClassName: "panel-collapse collapse in",
				Role:      "tabpanel",
			},
			r.Div(
				&r.DivProps{ClassName: "panel-body" + collapse},
				r.Div(nil, imgs...),
			),
		),
	)
}

type expandClick struct {
	ch *ChooserDef
	i  int
}

func (ex expandClick) OnClick(e *r.SyntheticMouseEvent) {
	s := ex.ch.State()
	s.open = ex.i
	ex.ch.SetState(s)

	e.PreventDefault()
}

type shuffleClick struct{ *ChooserDef }

func (sh shuffleClick) OnClick(e *r.SyntheticMouseEvent) {
	c := sh.ChooserDef.Props().Config

	var parts []string

	for _, cat := range c.Categories {
		p := cat.Options[rand.Intn(len(cat.Options))]
		parts = append(parts, p)
	}

	g := &Gopher{Parts: parts}

	sh.Props().Update.UpdateGopher(g)
}

func randElem(ss []string) string {
	return ss[rand.Intn(len(ss))]
}

type resetClick struct{ *ChooserDef }

func (sh resetClick) OnClick(e *r.SyntheticMouseEvent) {
	sh.Props().Update.UpdateGopher(nil)
}
