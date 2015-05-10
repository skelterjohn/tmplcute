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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const showStackTrace = true

func considerCaseOverwrite(t *testing.T, foo func() map[string]interface{}, input interface{}, expected map[string]interface{}) {
	errf := func(k string, v, e interface{}) {
		es := fmt.Sprintf("%q", e)
		if e == nil {
			es = "nil"
		}
		t.Errorf("for %q, got %s %q, expected %s", input, k, v, es)
	}

	var result map[string]interface{}

	if !showStackTrace {
		defer func() {
			panicMsg := recover()
			expectedPanicMsg, ok := expected["panic"]
			if panicMsg == nil != !ok {
				errf("panic", panicMsg, expectedPanicMsg)
				return
			}
			if !reflect.DeepEqual(panicMsg, expectedPanicMsg) {
				errf("panic", panicMsg, expectedPanicMsg)
				return
			}
		}()
	}
	result = foo()
	if len(result) != len(expected) {
		for key, expectedValue := range expected {
			value, ok := result[key]
			if !ok {
				errf(key, value, expectedValue)
			}
		}
	}
	for key, value := range result {
		expectedValue, ok := expected[key]
		if !ok || !reflect.DeepEqual(value, expectedValue) {
			errf(key, value, expectedValue)
		}
	}
}

type T1 struct {
	SS []S1
}

type T2 struct {
	V string
}

type S1 []T2

func apply(initial, delta string, ref interface{}) (string, error) {
	err := json.NewDecoder(strings.NewReader(initial)).Decode(ref)
	if err != nil {
		return "", err
	}
	err = json.NewDecoder(strings.NewReader(delta)).Decode(ref)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(ref)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func TestParseKey(t *testing.T) {
	type Case struct {
		Keystr   string
		Expected map[string]interface{}
	}

	cases := []Case{
		{
			Keystr: "x.y.z",
			Expected: map[string]interface{}{
				"key": fieldKey{
					field: "x",
					subkey: fieldKey{
						field: "y",
						subkey: fieldKey{
							field:  "z",
							subkey: terminalKey{},
						},
					},
				},
			},
		},
		{
			Keystr: "x",
			Expected: map[string]interface{}{
				"key": fieldKey{
					field:  "x",
					subkey: terminalKey{},
				},
			},
		},
		{
			Keystr: "x[2]",
			Expected: map[string]interface{}{
				"key": fieldKey{
					field: "x",
					subkey: indexKey{
						index:  2,
						subkey: terminalKey{},
					},
				},
			},
		},
		{
			Keystr: "[2]",
			Expected: map[string]interface{}{
				"key": indexKey{
					index:  2,
					subkey: terminalKey{},
				},
			},
		},
		{
			Keystr: "x[2].y",
			Expected: map[string]interface{}{
				"key": fieldKey{
					field: "x",
					subkey: indexKey{
						index: 2,
						subkey: fieldKey{
							field:  "y",
							subkey: terminalKey{},
						},
					},
				},
			},
		},
	}
	for _, c := range cases {
		foo := func() map[string]interface{} {
			key, err := parseKey(c.Keystr)
			result := map[string]interface{}{}
			if key != nil {
				result["key"] = key
			}
			if err != nil {
				result["err"] = err.Error()
			}
			return result
		}
		considerCaseOverwrite(t, foo, c.Keystr, c.Expected)
	}
}

func TestApplyKey(t *testing.T) {
	type Case struct {
		Keystr   string
		Value    string
		Obj      interface{}
		Expected map[string]interface{}
	}

	newInterfacePointer := func() *interface{} {
		var i interface{}
		return &i
	}

	cases := []Case{
		{
			Keystr: "V1",
			Value:  "foo",
			Obj:    &WT1{},
			Expected: map[string]interface{}{
				"obj": &WT1{
					V1: "foo",
				},
			},
		},
		{
			Keystr: "f1.v1",
			Value:  "foo",
			Obj:    &WT2{},
			Expected: map[string]interface{}{
				"obj": &WT2{
					F1: WT1{
						V1: "foo",
					},
				},
			},
		},
		{
			Keystr: "vs[1]",
			Value:  "foo",
			Obj:    &WT3{},
			Expected: map[string]interface{}{
				"obj": &WT3{
					Vs: []string{"", "foo"},
				},
			},
		},
		{
			Keystr: "vs[1]",
			Value:  "foo",
			Obj: &WT3{
				Vs: []string{"x"},
			},
			Expected: map[string]interface{}{
				"obj": &WT3{
					Vs: []string{"x", "foo"},
				},
			},
		},
		{
			Keystr: "vs[1]",
			Value:  "foo",
			Obj: &WT3{
				Vs: []string{"x", "y", "z"},
			},
			Expected: map[string]interface{}{
				"obj": &WT3{
					Vs: []string{"x", "foo", "z"},
				},
			},
		},
		{
			Keystr: "f1.vs[1]",
			Value:  "foo",
			Obj: &WT4{
				F1: WT3{
					Vs: []string{"x", "y", "z"},
				},
			},
			Expected: map[string]interface{}{
				"obj": &WT4{
					F1: WT3{
						Vs: []string{"x", "foo", "z"},
					},
				},
			},
		},
		{
			Keystr: "m1.foo.v1",
			Value:  "bar",
			Obj: &WT5{
				M1: map[string]WT1{},
			},
			Expected: map[string]interface{}{
				"obj": &WT5{
					M1: map[string]WT1{
						"foo": WT1{
							V1: "bar",
						},
					},
				},
			},
		},
		{
			Keystr: "f1.bar",
			Value:  "baz",
			Obj:    &WT6{},
			Expected: map[string]interface{}{
				"obj": &WT6{
					F1: map[string]interface{}{
						"bar": "baz",
					},
				},
			},
		},
		{
			Keystr: "f1",
			Value:  "bar",
			Obj:    &WT6{},
			Expected: map[string]interface{}{
				"obj": &WT6{
					F1: "bar",
				},
			},
		},
		{
			Keystr: "m1.bar",
			Value:  "baz",
			Obj:    &WT7{},
			Expected: map[string]interface{}{
				"obj": &WT7{
					M1: map[string]interface{}{
						"bar": "baz",
					},
				},
			},
		},
		{
			Keystr: "k1",
			Value:  "foo",
			Obj:    map[string]interface{}{},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k1": "foo",
				},
			},
		},
		{
			Keystr: "k1.k2",
			Value:  "foo",
			Obj:    map[string]interface{}{},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k1": map[string]interface{}{
						"k2": "foo",
					},
				},
			},
		},
		{
			Keystr: "k1.k2",
			Value:  "foo",
			Obj: map[string]interface{}{
				"k3": "bar",
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k3": "bar",
					"k1": map[string]interface{}{
						"k2": "foo",
					},
				},
			},
		},
		{
			Keystr: "Name",
			Value:  "Tuesday",
			Obj: map[string]interface{}{
				"Name": "Wednesday",
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"Name": "Tuesday",
				},
			},
		},
		{
			Keystr: "Parents[0]",
			Value:  "Fester",
			Obj: map[string]interface{}{
				"Name": "Wednesday",
				"Parents": []string{
					"Gomez",
					"Morticia",
				},
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"Name": "Wednesday",
					"Parents": []string{
						"Fester",
						"Morticia",
					},
				},
			},
		},
		{
			Keystr: "Parents[0]",
			Value:  "Fester",
			Obj: map[string]interface{}{
				"Name": "Wednesday",
				"Parents": []interface{}{
					"Gomez",
					"Morticia",
				},
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"Name": "Wednesday",
					"Parents": []interface{}{
						"Fester",
						"Morticia",
					},
				},
			},
		},
		{
			Keystr: "Parents[0]",
			Value:  "Fester",
			Obj: map[string]interface{}{
				"Name": "Wednesday",
				"Parents": []interface{}{
					nil,
					"Morticia",
				},
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"Name": "Wednesday",
					"Parents": []interface{}{
						"Fester",
						"Morticia",
					},
				},
			},
		},
		{
			Keystr: "Name2",
			Value:  "Tuesday",
			Obj: map[string]interface{}{
				"Name": "Wednesday",
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"Name":  "Wednesday",
					"Name2": "Tuesday",
				},
			},
		},
		{
			Keystr: "k1.k2",
			Value:  "foo",
			Obj: map[string]interface{}{
				"k1": map[string]interface{}{
					"k2": "bar",
				},
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k1": map[string]interface{}{
						"k2": "foo",
					},
				},
			},
		},
		{
			Keystr: "k1.v1",
			Value:  "foo",
			Obj: map[string]interface{}{
				"k1": WT1{
					V1: "bar",
				},
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k1": WT1{
						V1: "foo",
					},
				},
			},
		},
		{
			Keystr: "k1",
			Value:  "foo",
			Obj: map[string]interface{}{
				"k1": nil,
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k1": "foo",
				},
			},
		},
		{
			Keystr: "k1[0].k2",
			Value:  "foo",
			Obj: map[string]interface{}{
				"k1": nil,
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k1": []interface{}{
						map[string]interface{}{
							"k2": "foo",
						},
					},
				},
			},
		},
		{
			Keystr: "k1[0].k2",
			Value:  "foo",
			Obj: map[string]interface{}{
				"k1": []interface{}{},
			},
			Expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"k1": []interface{}{
						map[string]interface{}{
							"k2": "foo",
						},
					},
				},
			},
		},
	}
	for _, c := range cases {
		foo := func() map[string]interface{} {
			obj := c.Obj
			err := Overwrite(obj, c.Keystr, c.Value)
			result := map[string]interface{}{}
			if err == nil {
				result["obj"] = obj
			}
			if err != nil {
				result["err"] = err.Error()
			}
			return result
		}
		considerCaseOverwrite(t, foo, c.Keystr, c.Expected)
	}
	_ = newInterfacePointer
}

type WT1 struct {
	V1 string
}

type WT2 struct {
	F1 WT1
}

type WT3 struct {
	Vs []string
}

type WT4 struct {
	F1 WT3
}

type WT5 struct {
	M1 map[string]WT1
}

type WT6 struct {
	F1 interface{}
}

type WT7 struct {
	M1 map[string]interface{}
}
