package main

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// Reads all .txt files in the current folder
// and encodes them as strings literals in textfiles.go
func main() {
	fs, _ := ioutil.ReadDir("luasrc")
	out, _ := os.Create("luasrc/lua-generated.go")
	out.Write([]byte("package luasrc \n\nconst (\n"))
	for _, f := range fs {
		if strings.HasSuffix(f.Name(), ".lua") {
			name := strings.TrimSuffix(f.Name(), ".lua")
			name = strings.ToUpper(string(name[0])) + name[1:]
			out.Write([]byte(name + " = `"))
			f, _ := os.Open("luasrc/" + f.Name())
			io.Copy(out, f)
			out.Write([]byte("`\n"))
		}
	}
	out.Write([]byte(")\n"))
}
