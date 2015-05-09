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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

func orExit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	obj := map[string]interface{}{}
	for _, arg := range os.Args[1:] {
		processArg(arg, obj)
	}

	data, err := ioutil.ReadAll(os.Stdin)
	orExit(err)

	tmpl, err := template.New("tmplcute").Parse(string(data))
	orExit(err)

	orExit(tmpl.Execute(os.Stdout, obj))
}

func processArg(arg string, obj interface{}) {
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
		panic("not implemented")
	}
	if strings.HasSuffix(strings.ToLower(arg), ".yaml") {
		panic("not implemented")
	}
	if strings.HasSuffix(strings.ToLower(arg), ".rjson") {
		panic("not implemented")
	}
	fmt.Fprintf(os.Stderr, "don't know what to do with %q\n", arg)
	os.Exit(1)
}
