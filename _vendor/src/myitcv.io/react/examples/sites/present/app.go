// Template generated by reactGen

package main

import (
	"bytes"
	"fmt"
	"strings"

	"honnef.co/go/js/dom"
	"honnef.co/go/js/xhr"
	"myitcv.io/react"
)

type AppDef struct {
	react.ComponentDef
}

type AppState struct {
	URL    string
	Slides string
}

func App() *AppElem {
	return buildAppElem()
}

func (a AppDef) ComponentWillMount() {
	go func() {
		if err := initTemplates("."); err != nil {
			panic(err)
		}
	}()
}

func (a AppDef) Render() react.Element {
	s := a.State()

	var iframe react.Element

	if s.Slides != "" {
		iframe = react.IFrame(&react.IFrameProps{
			SrcDoc: s.Slides,
			Style: &react.CSS{
				Width:    "100%",
				Height:   "100%",
				Position: "absolute",
				Top:      "0px",
				Left:     "0px",
				ZIndex:   "1",
				Overflow: "hidden",
			},
		})
	} else {
		iframe = react.H1(&react.H1Props{
			Style: &react.CSS{
				Position: "absolute",
				Top:      "50%",
				Left:     "50%",
			},
		},
			react.S("Enter a URL to start"),
		)
	}

	return react.Div(
		&react.DivProps{
			Style: &react.CSS{
				Overflow: "hidden",
			},
		},
		react.Input(&react.InputProps{
			OnChange: urlChange{a},
			Value:    s.URL,
			Style: &react.CSS{
				Width:    "100%",
				Position: "absolute",
				Top:      "0px",
				Left:     "0px",
				ZIndex:   "2",
			},
		}),
		iframe,
	)
}

type urlChange struct{ AppDef }

func (i urlChange) OnChange(se *react.SyntheticEvent) {
	target := se.Target().(*dom.HTMLInputElement)
	u := target.Value

	st := i.State()
	st.URL = u
	i.SetState(st)

	if u == "" {
		return
	}

	go func() {
		req := xhr.NewRequest("GET", u)
		err := req.Send(nil)
		if err != nil {
			fmt.Printf("Failed to fetch %v\n", u)
			return
		}

		out := new(bytes.Buffer)
		in := strings.NewReader(req.ResponseText)

		err = renderDoc(out, u, in)
		if err != nil {
			panic(err)
		}

		st := i.State()
		st.Slides = out.String()
		i.SetState(st)
	}()
}
