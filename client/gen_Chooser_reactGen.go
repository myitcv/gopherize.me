// Code generated by reactGen. DO NOT EDIT.

package main

import "myitcv.io/react"

type ChooserElem struct {
	react.Element
}

func buildChooser(cd react.ComponentDef) react.Component {
	return ChooserDef{ComponentDef: cd}
}

func buildChooserElem(props ChooserProps, children ...react.Element) *ChooserElem {
	return &ChooserElem{
		Element: react.CreateElement(buildChooser, props, children...),
	}
}

func (c ChooserDef) RendersElement() react.Element {
	return c.Render()
}

// SetState is an auto-generated proxy proxy to update the state for the
// Chooser component.  SetState does not immediately mutate c.State()
// but creates a pending state transition.
func (c ChooserDef) SetState(state ChooserState) {
	c.ComponentDef.SetState(state)
}

// State is an auto-generated proxy to return the current state in use for the
// render of the Chooser component
func (c ChooserDef) State() ChooserState {
	return c.ComponentDef.State().(ChooserState)
}

// IsState is an auto-generated definition so that ChooserState implements
// the myitcv.io/react.State interface.
func (c ChooserState) IsState() {}

var _ react.State = ChooserState{}

// GetInitialStateIntf is an auto-generated proxy to GetInitialState
func (c ChooserDef) GetInitialStateIntf() react.State {
	return ChooserState{}
}

func (c ChooserState) EqualsIntf(val react.State) bool {
	return c == val.(ChooserState)
}

// IsProps is an auto-generated definition so that ChooserProps implements
// the myitcv.io/react.Props interface.
func (c ChooserProps) IsProps() {}

// Props is an auto-generated proxy to the current props of Chooser
func (c ChooserDef) Props() ChooserProps {
	uprops := c.ComponentDef.Props()
	return uprops.(ChooserProps)
}

func (c ChooserProps) EqualsIntf(val react.Props) bool {
	return c == val.(ChooserProps)
}

var _ react.Props = ChooserProps{}
