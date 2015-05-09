# tmplcute #

## Simple CLI for exercising Go's text/template ##

tmplcute - exercise go's text/template
```
Usage: tmplcute [-h] [ --KEY=VALUE | FILE.json | FILE.yaml ]*
```

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
