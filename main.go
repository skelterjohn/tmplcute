/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

func orExit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var usage = `tmplcute - exercise go's text/template
Usage: tmplcute [-h] [ --KEY=VALUE | FILE.json | FILE.yaml ]*

tmplcute reads a text/template from stdin, and executes it onto stdout using
the object build by arguments.

KEY/VALUE pairs and FILEs are used to build up the object used for the
template's execution. The object begins life as a map[string]interface{}, and
each argument builds it up.

FILE.json and FILE.yaml decode the document onto the object.

--KEY=VALUE sets a value in the object, using KEY to index into it. The KEYs are
dotted and indexed. For example, "--foo.bar=baz" will create a 'foo' field if it
does not already exist, and then give it a 'bar' field with the value "baz". Or,
"--arr[0]=123" will create an 'arr' field that is a slice, and set its first
element to the string "123" if it does not already exist, or attempt to match
its type if it does (types may already have been set by the other decoders).
`

func main() {
	obj := map[string]interface{}{}
	for _, arg := range os.Args[1:] {
		processArg(arg, &obj)
	}

	data, err := ioutil.ReadAll(os.Stdin)
	orExit(err)

	tmpl, err := template.New("tmplcute").Parse(string(data))
	orExit(err)

	orExit(tmpl.Execute(os.Stdout, obj))
}

func processArg(arg string, obj interface{}) {
	if arg == "-h" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if strings.HasPrefix(arg, "--") {
		keyval := arg[2:]
		tokens := strings.SplitN(keyval, "=", 2)
		if len(tokens) != 2 {
			fmt.Fprintf(os.Stderr, "value for %q must be in the form of %q\n", arg, arg+"=VALUE")
			os.Exit(1)
		}
		key, val := tokens[0], tokens[1]
		orExit(Overwrite(obj, key, val))
		return
	}
	if strings.HasSuffix(strings.ToLower(arg), ".json") {
		fin, err := os.Open(arg)
		orExit(err)
		orExit(json.NewDecoder(fin).Decode(obj))
		return
	}
	if strings.HasSuffix(strings.ToLower(arg), ".yaml") {
		fin, err := os.Open(arg)
		orExit(err)
		data, err := ioutil.ReadAll(fin)
		orExit(err)
		orExit(yaml.Unmarshal(data, obj))
		return
	}
	if strings.HasSuffix(strings.ToLower(arg), ".rjson") {
		fmt.Fprintln(os.Stderr, ".rjson support is not implemnented")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "don't know what to do with %q\n", arg)
	os.Exit(1)
}
