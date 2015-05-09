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
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

/*
Overwrite a value in a nested object, using the key to find it.

The obj parameter must be a struct, a map with string keys, a slice or
array, or primitive data type (int, string, etc). All nested objects must also
be one of these types or pointers to one of these types.

The keystr is of the form "FIELD", "[INDEX]", or either of those followed by a
subkey, which is a key preceded by a ".". So, "x", "x.y[2].z", or "[2]" are
examples.
*/
func Overwrite(obj interface{}, keystr, value string) error {
	k, err := parseKey(keystr)
	if err != nil {
		return err
	}

	return k.apply(reflect.ValueOf(obj), value)
}

type key interface {
	String() string
	apply(obj reflect.Value, value string) error
}

type terminalKey struct {
}

func (terminalKey) String() string {
	return ""
}

/*
With terminalKey.apply(), the object has the value put into it directly; no
more subkeys.
*/
func (k terminalKey) apply(obj reflect.Value, value string) error {
	switch obj.Type().Kind() {
	case reflect.Ptr:
		if obj.IsNil() {
			obj.Set(reflect.New(obj.Type().Elem()))
		}
		return k.apply(obj.Elem(), value)
	case reflect.Interface:
		obj.Set(reflect.ValueOf(value))
	case reflect.String:
		obj.SetString(value)
	case reflect.Float32, reflect.Float64:
		fvalue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		obj.SetFloat(fvalue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ivalue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		obj.SetInt(ivalue)
	case reflect.Bool:
		bvalue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		obj.SetBool(bvalue)
	default:
		// types that can't take simple values will fall through to here. includes
		// maps, slices, etc.
		return fmt.Errorf("don't know how to set value type %T", obj.Interface())
	}
	return nil
}

type fieldKey struct {
	field  string
	subkey key
}

func (k fieldKey) String() string {
	switch k.subkey.(type) {
	case fieldKey:
		return k.field + "." + k.subkey.String()
	default:
		return k.field + k.subkey.String()
	}
}

func (k fieldKey) apply(obj reflect.Value, value string) error {
	switch obj.Type().Kind() {
	case reflect.Interface:
		// apply to the contents of the interface.
		// TODO(jasmuth): what if there is nothing in this interface?
		return k.apply(obj.Elem(), value)
	case reflect.Ptr:
		// pointers need to be dereferenced and, if necessary, allocated
		if obj.IsNil() {
			obj.Set(reflect.New(obj.Type().Elem()))
		}
		return k.apply(obj.Elem(), value)
	case reflect.Map:
		return k.applyToMap(obj, value)
	case reflect.Struct:
		return k.applyToStruct(obj, value)
	default:
		return fmt.Errorf("%s: expected map or struct, got %T", k, obj.Interface())
	}
}

// The subkeyType() function is used to choose types when none is specified
// by the input. When overwriting into an interface{}, a concrete type is
// chosen here. It will be a map[string]interface{}, []interface{}, or string
// depending on whether the subkey is a field, index, or terminal.
func (k fieldKey) subkeyType() reflect.Type {
	var i interface{}
	switch k.subkey.(type) {
	case fieldKey:
		return reflect.MapOf(reflect.TypeOf("string"), reflect.TypeOf(&i).Elem())
	case indexKey:
		return reflect.SliceOf(reflect.TypeOf(&i).Elem())
	case terminalKey:
		return reflect.TypeOf("")
	default:
		panic("unreachable")
	}
}

func (k fieldKey) applyToMap(obj reflect.Value, value string) error {
	// if the map is not allocated, do that
	if obj.IsNil() {
		obj.Set(reflect.MakeMap(obj.Type()))
	}

	if obj.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("%s: expected map with string keys, got map with %s keys", k, obj.Type().Key().Name())
	}

	// iterate through keys because we want to not be case sensitive
	for _, mapKey := range obj.MapKeys() {
		strkey := mapKey.Interface().(string)
		if strings.ToLower(strkey) != strings.ToLower(k.field) {
			continue
		}
		subObj := obj.MapIndex(mapKey)
		var origObj reflect.Value
		if subObj.Type().Kind() == reflect.Interface {
			if subObj.IsNil() {
				// if the key is in here but points to a nil interface, guess the
				// type of the thing to replace it with.
				origObj = reflect.New(k.subkeyType()).Elem()
			} else {
				// otherwise dereference the interface so we can write into that value.
				origObj = subObj.Elem()
			}
		} else {
			// if it isn't an interface, we can begin immediately.
			origObj = subObj
		}

		// since map values aren't addressable, we need to make a copy and
		// reinsert it back into the map.
		var copyObj reflect.Value
		switch origObj.Kind() {
		case reflect.Slice:
			// if it's a slice, make sure we copy the contents before overwriting.
			sliceObj := reflect.MakeSlice(origObj.Type(), origObj.Len(), origObj.Cap())
			reflect.Copy(sliceObj, origObj)
			copyObj = reflect.New(sliceObj.Type()).Elem()
			copyObj.Set(sliceObj)
		default:
			copyObj = reflect.New(origObj.Type()).Elem()
		}

		err := k.subkey.apply(copyObj, value)
		if err != nil {
			return err
		}
		obj.SetMapIndex(mapKey, copyObj)
		return nil
	}

	// key not found, need to make a new one
	concreteType := obj.Type().Elem()
	if concreteType.Kind() == reflect.Interface {
		concreteType = k.subkeyType()
	}
	concreteValue := reflect.New(concreteType).Elem()
	err := k.subkey.apply(concreteValue, value)
	if err != nil {
		return err
	}
	obj.SetMapIndex(reflect.ValueOf(k.field), concreteValue)
	return nil
}

func (k fieldKey) applyToStruct(obj reflect.Value, value string) error {
	// iterate through fields so that we can ignore case
	for i := 0; i < obj.NumField(); i++ {
		tf := obj.Type().Field(i)
		if strings.ToLower(tf.Name) != strings.ToLower(k.field) {
			continue
		}
		subObj := obj.Field(i)

		// instantiate it if necessary
		if subObj.Type().Kind() == reflect.Ptr && subObj.IsNil() {
			subObj.Set(reflect.New(subObj.Type()).Elem())
		}

		// if the field is a nil interface, make something new
		if subObj.Type().Kind() == reflect.Interface && subObj.IsNil() {

			// TODO(jasmuth): this next block can probably be simplified.

			// we aren't using subkeyType() here because the terminalKey
			// case is more complicated.
			switch k.subkey.(type) {
			case fieldKey:
				subObj.Set(reflect.ValueOf(map[string]interface{}{}))
			case indexKey:
				subObj.Set(reflect.ValueOf([]interface{}{}))
			case terminalKey:
				var concreteType reflect.Type
				// if it's an interface, use a string. if it's not, we use
				// the original type.
				if subObj.Type().Kind() == reflect.Interface {
					concreteType = reflect.TypeOf("")
				} else {
					concreteType = subObj.Type().Elem()
				}
				concreteObj := reflect.New(concreteType).Elem()
				err := k.subkey.apply(concreteObj, value)
				if err != nil {
					return err
				}
				subObj.Set(concreteObj)
				return nil
			}
		}

		return k.subkey.apply(subObj, value)
	}
	return fmt.Errorf("%s: no field %s for type %s", k, k.field, obj.Type().Name())
}

type indexKey struct {
	index  int
	subkey key
}

func (k indexKey) String() string {
	indexStr := fmt.Sprintf("[%d]", k.index)
	switch k.subkey.(type) {
	case fieldKey:
		return indexStr + "." + k.subkey.String()
	default:
		return indexStr + k.subkey.String()
	}
}

func (k indexKey) apply(obj reflect.Value, value string) error {
	if obj.Type().Kind() != reflect.Slice && obj.Type().Kind() != reflect.Array {
		return fmt.Errorf("%s: expected slice or array, got %s", k, obj.Type().Name())
	}

	if obj.Len() <= k.index && obj.Type().Kind() == reflect.Array {
		return fmt.Errorf("%s: array index too large: %d >= %d, got ", k, k.index, obj.Len())
	}

	// check if we need to grow the slice
	if obj.Len() <= k.index {
		newslice := reflect.MakeSlice(obj.Type(), k.index+1, k.index+1)
		reflect.Copy(newslice, obj)
		obj.Set(newslice)
	}

	subObj := obj.Index(k.index)
	return k.subkey.apply(subObj, value)
}

const subkeyPattern = `((?:\.|\[).+)?$`

var fieldKeyRE = regexp.MustCompile(`^\.?([a-zA-Z][a-zA-Z0-9]*)` + subkeyPattern)
var indexKeyRE = regexp.MustCompile(`^\[([0-9]+)\]` + subkeyPattern)

func parseKey(keystr string) (key, error) {
	if groups := fieldKeyRE.FindAllStringSubmatch(keystr, 1); groups != nil {
		var k fieldKey
		var err error
		k.field = groups[0][1]
		subkeystr := groups[0][2]
		if len(subkeystr) != 0 {
			k.subkey, err = parseKey(subkeystr)
			if err != nil {
				return nil, err
			}
		} else {
			k.subkey = terminalKey{}
		}
		return k, nil
	}
	if groups := indexKeyRE.FindAllStringSubmatch(keystr, 1); groups != nil {
		var k indexKey
		var err error
		k.index, err = strconv.Atoi(groups[0][1])
		if err != nil {
			return nil, err
		}
		subkeystr := groups[0][2]
		if len(subkeystr) != 0 {
			k.subkey, err = parseKey(subkeystr)
			if err != nil {
				return nil, err
			}
		} else {
			k.subkey = terminalKey{}
		}
		return k, nil
	}
	return nil, fmt.Errorf("invalid key: %q", keystr)
}
