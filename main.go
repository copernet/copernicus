// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
)

type A struct {
	a B
}

func (a *A) t(){
	fmt.Println("a.t()")
}
type B struct {
	b int
}

func (a *B) t(){
	fmt.Println("b.t()")
}

func main() {
	ta := new(A)
	taa := *ta
	ta.t()
	taa.t()
}