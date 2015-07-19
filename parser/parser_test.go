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
3]]
var test  = 32
`
	comments := String(s)
	fmt.Printf("R: 2 == %v, Comment 1 = %v\n", comments[0].Row, comments[0].Value)
	fmt.Printf("R: 4 == %v, Comment 1 = %v\n", comments[1].Row, comments[1].Value)
	fmt.Printf("R: 7 == %v, Comment 1 = %v\n", comments[2].Row, comments[2].Value)
	fmt.Printf("R: 11 == %v, Comment 1 = %v\n", comments[3].Row, comments[3].Value)
}
