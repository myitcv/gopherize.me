// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"fmt"

	r "myitcv.io/react"

	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document()

func main() {
	domTarget := document.GetElementByID("gopherize.me")

	r.Render(Outer(), domTarget)
}

const (
	debug = true
)

func debugf(format string, args ...interface{}) {
	if debug {
		fmt.Printf(format, args...)
	}
}

func debugln(args ...interface{}) {
	if debug {
		fmt.Println(args...)
	}
}
