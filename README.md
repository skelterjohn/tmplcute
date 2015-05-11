# tmplcute (template execute) #

## Simple CLI for exercising Go's text/template ##

tmplcute - exercise go's text/template
```
Usage: tmplcute [-h] [-w] [ --KEY=VALUE | FILE{.json,.rjson,.yaml} ]*
```
tmplcute reads a text/template from stdin, and executes it onto stdout using
the object build by arguments.

The "-w" flag indicates that "html/template" should be used rather than the
normal "text/template".

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

The templating also has embedded funcs for output in json, rjson, or yaml.

## Examples ##
fields
```
$ echo '{{.foo}}' | tmplcute --foo=bar
bar
```
slices
```
$ echo '{{range .foo}}{{.}}{{end}}' | tmplcute --foo[0]=abc --foo[1]=xyz
abcxyz
```
nested fields
```
$ echo '{{.x.y}}' | tmplcute --x.y=z
z
```
fields nested under arrays
```
$ echo '{{range .arr}}{{.x}},{{end}}' | tmplcute --arr[0].x=y --arr[1].x=z
y,z,
```
input from document files
```
$ cat examples/data.json 
{"foo": "bar"}
$ cat examples/data.yaml 
foo: bar
$ echo '{{.foo}}' | tmplcute examples/data.json
bar
$ echo '{{.foo}}' | tmplcute examples/data.yaml 
bar
```
overriding values from previous args
```
$ cat > examples/twothings.yaml
a: b
c: d
$ echo '{{.a}} {{.c}}' | tmplcute examples/twothings.yaml 
b d
$ echo '{{.a}} {{.c}}' | tmplcute examples/twothings.yaml --c=e
b e
```
avoid script injections
```
$ echo 'welcome {{.name}}!' | tmplcute -w --name='<script>do js nonsense</script>'
welcome &lt;script&gt;do js nonsense&lt;/script&gt;!
```
output rjson (can swap in json or yaml)
```
$ echo '{{rjson .}}' | tmplcute --x[0]=z
{
  x: [
    "z"
  ]
}
```
