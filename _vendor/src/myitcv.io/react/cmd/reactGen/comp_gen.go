package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"strings"
	"unicode"
	"unicode/utf8"

	"myitcv.io/gogenerate"
)

type compGen struct {
	*coreGen

	Recv string
	Name string

	HasState                     bool
	HasProps                     bool
	HasGetInitState              bool
	HasComponentWillReceiveProps bool

	PropsHasEquals bool
	StateHasEquals bool
}

func (g *gen) genComp(defName string) {

	name := strings.TrimSuffix(defName, compDefSuffix)

	if g.isReactCore {
		panic(fmt.Errorf("don't yet know how to generate core components like %v", name))
	}

	r, _ := utf8.DecodeRuneInString(name)

	cg := &compGen{
		coreGen: newCoreGen(g),
		Name:    name,
		Recv:    string(unicode.ToLower(r)),
	}

	_, hasState := g.types[name+stateTypeSuffix]

	_, hasPropsTmpl := g.types[propsTypeTmplPrefix+name+propsTypeSuffix]
	_, hasPropsType := g.types[name+propsTypeSuffix]

	hasProps := hasPropsTmpl || hasPropsType

	cg.HasState = hasState
	cg.HasProps = hasProps

	if hasState {
		for _, ff := range g.nonPointMeths[defName] {
			m := ff.fn

			if m.Name.Name != getInitialState {
				continue
			}

			if m.Type.Params != nil && len(m.Type.Params.List) > 0 {
				continue
			}

			if m.Type.Results != nil && len(m.Type.Results.List) != 1 {
				continue
			}

			rp := m.Type.Results.List[0]

			id, ok := rp.Type.(*ast.Ident)
			if !ok {
				continue
			}

			if id.Name == name+stateTypeSuffix {
				cg.HasGetInitState = true
				break
			}
		}

		for _, ff := range g.nonPointMeths[name+stateTypeSuffix] {
			m := ff.fn

			if m.Name.Name != equals {
				continue
			}

			if m.Type.Params != nil && len(m.Type.Params.List) != 1 {
				continue
			}

			if m.Type.Results != nil && len(m.Type.Results.List) != 1 {
				continue
			}

			{
				v := m.Type.Params.List[0]

				id, ok := v.Type.(*ast.Ident)
				if !ok {
					continue
				}

				if id.Name != name+stateTypeSuffix {
					continue
				}
			}

			{
				v := m.Type.Results.List[0]

				id, ok := v.Type.(*ast.Ident)
				if !ok {
					continue
				}

				if id.Name != "bool" {
					continue
				}
			}

			cg.StateHasEquals = true
		}
	}

	if hasProps {
		for _, ff := range g.nonPointMeths[defName] {
			m := ff.fn

			if m.Name.Name != componentWillReceiveProps {
				continue
			}

			if m.Type.Params != nil && len(m.Type.Params.List) != 1 {
				continue
			}

			if m.Type.Results != nil && len(m.Type.Results.List) != 0 {
				continue
			}

			p := m.Type.Params.List[0]

			id, ok := p.Type.(*ast.Ident)
			if !ok {
				continue
			}

			if id.Name == name+propsTypeSuffix {
				cg.HasComponentWillReceiveProps = true
				break
			}
		}

		for _, ff := range g.nonPointMeths[name+propsTypeSuffix] {
			m := ff.fn

			if m.Name.Name != equals {
				continue
			}

			if m.Type.Params != nil && len(m.Type.Params.List) != 1 {
				continue
			}

			if m.Type.Results != nil && len(m.Type.Results.List) != 1 {
				continue
			}

			{
				v := m.Type.Params.List[0]

				id, ok := v.Type.(*ast.Ident)
				if !ok {
					continue
				}

				if id.Name != name+propsTypeSuffix {
					continue
				}
			}

			{
				v := m.Type.Results.List[0]

				id, ok := v.Type.(*ast.Ident)
				if !ok {
					continue
				}

				if id.Name != "bool" {
					continue
				}
			}

			cg.PropsHasEquals = true
		}
	}

	cg.pf("// Code generated by %v. DO NOT EDIT.\n", reactGenCmd)
	cg.pln()
	cg.pf("package %v\n", cg.pkg)

	cg.pf("import \"%v\"\n", reactPkg)
	cg.pln()

	cg.pt(`
type {{.Name}}Elem struct {
	react.Element
}

func ({{.Recv}} {{.Name}}Def) ShouldComponentUpdateIntf(nextProps react.Props, prevState, nextState react.State) bool {
	res := false

	{{if .HasProps -}}
	{
	{{if .PropsHasEquals -}}
	res = !{{.Recv}}.Props().Equals(nextProps.({{.Name}}Props)) || res
	{{else -}}
	res = {{.Recv}}.Props() != nextProps.({{.Name}}Props) || res
	{{end -}}
	}
	{{end -}}
	{{if .HasState -}}
	v := prevState.({{.Name}}State)
	res = !v.EqualsIntf(nextState) || res
	{{end -}}

	return res
}

func build{{.Name}}(cd react.ComponentDef) react.Component {
	return {{.Name}}Def{ComponentDef: cd}
}

func build{{.Name}}Elem({{if .HasProps}}props {{.Name}}Props,{{end}} children ...react.Element) *{{.Name}}Elem {
	return &{{.Name}}Elem{
		Element: react.CreateElement(build{{.Name}}, {{if .HasProps}}props{{else}}nil{{end}}),
	}
}

{{if .HasState}}
// SetState is an auto-generated proxy proxy to update the state for the
// {{.Name}} component.  SetState does not immediately mutate {{.Recv}}.State()
// but creates a pending state transition.
func ({{.Recv}} {{.Name}}Def) SetState(state {{.Name}}State) {
	{{.Recv}}.ComponentDef.SetState(state)
}

// State is an auto-generated proxy to return the current state in use for the
// render of the {{.Name}} component
func ({{.Recv}} {{.Name}}Def) State() {{.Name}}State {
	return {{.Recv}}.ComponentDef.State().({{.Name}}State)
}

// IsState is an auto-generated definition so that {{.Name}}State implements
// the myitcv.io/react.State interface.
func ({{.Recv}} {{.Name}}State) IsState() {}

var _ react.State = {{.Name}}State{}

// GetInitialStateIntf is an auto-generated proxy to GetInitialState
func ({{.Recv}} {{.Name}}Def) GetInitialStateIntf() react.State {
{{if .HasGetInitState -}}
	return {{.Recv}}.GetInitialState()
{{else -}}
	return {{.Name}}State{}
{{end -}}
}

func ({{.Recv}} {{.Name}}State) EqualsIntf(val react.State) bool {
	{{if .StateHasEquals -}}
	return {{.Recv}}.Equals(val.({{.Name}}State))
	{{else -}}
	return {{.Recv}} == val.({{.Name}}State)
	{{end -}}
}
{{end}}


{{if .HasProps}}
// IsProps is an auto-generated definition so that {{.Name}}Props implements
// the myitcv.io/react.Props interface.
func ({{.Recv}} {{.Name}}Props) IsProps() {}

// Props is an auto-generated proxy to the current props of {{.Name}}
func ({{.Recv}} {{.Name}}Def) Props() {{.Name}}Props {
	uprops := {{.Recv}}.ComponentDef.Props()
	return uprops.({{.Name}}Props)
}

{{if .HasComponentWillReceiveProps}}
// ComponentWillReceivePropsIntf is an auto-generated proxy to
// ComponentWillReceiveProps
func ({{.Recv}} {{.Name}}Def) ComponentWillReceivePropsIntf(val interface{}) {
	ourProps := val.({{.Name}}Props)
	{{.Recv}}.ComponentWillReceiveProps(ourProps)
}
{{end}}

func ({{.Recv}} {{.Name}}Props) EqualsIntf(val react.Props) bool {
	{{if .PropsHasEquals -}}
	return {{.Recv}}.Equals(val.({{.Name}}Props))
	{{else -}}
	return {{.Recv}} == val.({{.Name}}Props)
	{{end -}}
}

var _ react.Props = {{.Name}}Props{}
{{end}}
	`, cg)

	ofName := gogenerate.NameFile(name, reactGenCmd)
	toWrite := cg.buf.Bytes()

	out, err := format.Source(toWrite)
	if err == nil {
		toWrite = out
	}

	wrote, err := gogenerate.WriteIfDiff(toWrite, ofName)
	if err != nil {
		fatalf("could not write %v: %v", ofName, err)
	}

	if wrote {
		infof("writing %v", ofName)
	} else {
		infof("skipping writing of %v; it's identical", ofName)
	}

}
