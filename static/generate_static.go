// Copyright 2019 The Reserve Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build ignore

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	fs, _ := ioutil.ReadDir(".")
	out, _ := os.Create("static_generated.go")

	self, _ := os.Open("generate_static.go")
	selfReader := bufio.NewReader(self)
	for {
		line, _ := selfReader.ReadString('\n')
		if !strings.HasPrefix(line, "//") {
			break
		}
		out.Write([]byte(line))
	}
	out.Write([]byte("\n"))

	out.Write([]byte("// Code generated by generate_static.go; DO NOT EDIT.\n\npackage static\n\n"))
	out.Write([]byte("import \"time\"\n\n"))

	now := time.Now()
	out.Write([]byte("var ModTime = time.Unix("))
	out.Write([]byte(fmt.Sprintf("%d, %d", 0, now.UnixNano())))
	out.Write([]byte(")\n\n"))

	for _, file := range fs {
		filename := file.Name()
		if strings.HasPrefix(filename, ".") || strings.HasSuffix(filename, ".go") {
			continue
		}
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		parts := strings.Split(filename, ".")
		parts = append(strings.Split(parts[0], "_"), parts[1:]...)
		for i := range parts {
			parts[i] = strings.Title(parts[i])
		}
		basename := strings.Join(parts, "")

		out.Write([]byte("const "))
		out.Write([]byte(strings.Title(basename)))
		out.Write([]byte(" = "))
		out.Write([]byte(strconv.Quote(string(content))))
		out.Write([]byte("\n"))
	}
}
