package parser

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	s := `
--comment

--comment2
function test

--comment3

--[[1
Multi2
3--]]
var test  = 32
`
	comments := String(s)
	fmt.Printf("R: 2 == %v, Comment 1 = %v\n", comments[0].row, comments[0].value)
	fmt.Printf("R: 4 == %v, Comment 1 = %v\n", comments[1].row, comments[1].value)
	fmt.Printf("R: 7 == %v, Comment 1 = %v\n", comments[2].row, comments[2].value)
	fmt.Printf("R: 11 == %v, Comment 1 = %v\n", comments[3].row, comments[3].value)
}
