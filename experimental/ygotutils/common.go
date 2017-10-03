package ygotutils

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"

	log "github.com/golang/glog"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	scpb "google.golang.org/genproto/googleapis/rpc/code"
	spb "google.golang.org/genproto/googleapis/rpc/status"
)

var (
	// debugLibrary controls the debugging output from the library data tree
	// traversal.
	debugLibrary = false
	// maxCharsPerLine is the maximum number of characters per line from
	// dbgPrintln and dbgSchema. Additional characters are truncated.
	maxCharsPerLine = 1000
	// maxValueStrLen is the maximum number of characters output from valueStr.
	maxValueStrLen = 50

	// statusOK indicates an OK Status.
	statusOK = spb.Status{Code: int32(scpb.Code_OK)}
)

// IsValueStruct reports whether v is a struct type.
func IsValueStruct(v reflect.Value) bool {
	return v.Kind() == reflect.Struct
}

// IsValueStructPtr reports whether v is a struct ptr type.
func IsValueStructPtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr && IsValueStruct(v.Elem())
}

// IsValueMap reports whether v is a map type.
func IsValueMap(v reflect.Value) bool {
	return v.Kind() == reflect.Map
}

// IsTypeStructPtr reports whether v is a struct ptr type.
func IsTypeStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

// IsTypeSlicePtr reports whether v is a slice ptr type.
func IsTypeSlicePtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Slice
}

// IsTypeMap reports whether v is a map type.
func IsTypeMap(t reflect.Type) bool {
	return t.Kind() == reflect.Map
}

// pathMatchesPrefix reports whether prefix is a prefix of path.
func pathMatchesPrefix(path *gpb.Path, prefix []string) bool {
	if len(path.GetElem()) < len(prefix) {
		return false
	}
	for i := range prefix {
		if prefix[i] != path.GetElem()[i].GetName() {
			return false
		}
	}

	return true
}

// trimGNMIPathPrefix returns path with the prefix trimmed. It returns the
// original path if the prefix does not fully match.
func trimGNMIPathPrefix(path *gpb.Path, prefix []string) *gpb.Path {
	if !pathMatchesPrefix(path, prefix) {
		return path
	}
	out := *path
	out.Elem = out.GetElem()[len(prefix):]
	return &out
}

// popGNMIPath returns the supplied GNMI path with the first path element
// removed. If the path is empty, it returns an empty path.
func popGNMIPath(path *gpb.Path) *gpb.Path {
	if len(path.GetElem()) == 0 {
		return path
	}
	return &gpb.Path{
		Origin: path.GetOrigin(),
		Elem:   path.GetElem()[1:],
	}
}

// pathStructTagKey returns the string label of the struct field sf when it is
// used in a YANG list. This is the last path element of the struct path tag.
func pathStructTagKey(f reflect.StructField) string {
	p, err := pathToSchema(f)
	if err != nil {
		log.Errorln("struct field %s does not have a path tag, bad schema?", f.Name)
		return ""
	}
	return p[len(p)-1]
}

// childSchema returns the schema for the struct field f, if f contains a valid
// path tag and the schema path is found in the schema tree. It returns an error
// if the struct tag is invalid, or nil if tag is valid but the schema is not
// found in the tree at the specified path.
// childSchema is for use in key values and does not support choice.
func childSchema(schema *yang.Entry, f reflect.StructField) (*yang.Entry, error) {
	pathTag, _ := f.Tag.Lookup("path")
	dbgPrintln("childSchema for schema %s, field %s, tag %s", schema.Name, f.Name, pathTag)
	if rootName, ok := f.Tag.Lookup("rootname"); ok {
		return schema.Dir[rootName], nil
	}
	p, err := pathToSchema(f)
	if err != nil {
		return nil, err
	}

	// Containers have the container schema name as the first element in the
	// path tag for each field e.g. System { Dns ... path: "system/dns"
	// Strip this off since the supplied schema already refers to the struct
	// schema element.
	if schema.IsContainer() && len(p) > 1 && p[0] == schema.Name {
		p = p[1:]
	}
	dbgPrintln("pathToSchema yields %v", p)
	// For empty path, return the parent schema.
	childSchema := schema
	foundSchema := true
	// Traverse the returned schema path to get the child schema.
	dbgPrint("traversing schema Dirs...")
	for ; len(p) > 0; p = p[1:] {
		dbgPrintNoIndent("/%s", p[0])
		ns, ok := childSchema.Dir[stripModulePrefix(p[0])]
		if !ok {
			foundSchema = false
			break
		}
		childSchema = ns
	}
	if foundSchema {
		dbgPrintNoIndent(" - found\n")
		return childSchema, nil
	}
	dbgPrintNoIndent(" - not found\n")

	return nil, nil
}

// pathToSchema returns a path to the schema for the struct field f.
// Paths are embedded in the "path" struct tag and can be either simple:
//   e.g. "path:a"
// or composite e.g.
//   e.g. "path:config/a|a"
// which is found in OpenConfig leaf-ref cases where the key of a list is a
// leafref. In the latter case, this function returns {"config", "a"}, and the
// schema *yang.Entry for the field is given by schema.Dir["config"].Dir["a"].
func pathToSchema(f reflect.StructField) ([]string, error) {
	pathAnnotation, ok := f.Tag.Lookup("path")
	if !ok {
		return nil, fmt.Errorf("field %s did not specify a path", f.Name)
	}

	paths := strings.Split(pathAnnotation, "|")
	if len(paths) == 1 {
		pathAnnotation = strings.TrimPrefix(pathAnnotation, "/")
		return strings.Split(pathAnnotation, "/"), nil
	}
	for _, pv := range paths {
		pv = strings.TrimPrefix(pv, "/")
		pe := strings.Split(pv, "/")
		if len(pe) > 1 {
			return pe, nil
		}
	}

	return nil, fmt.Errorf("field %s had path tag %s with |, but no elements of form a/b", f.Name, pathAnnotation)
}

// schemaPaths returns all the paths in the path tag, plus the path value of
// the rootname tag, if one is present.
func schemaPaths(f reflect.StructField) ([][]string, error) {
	var out [][]string
	rootTag, ok := f.Tag.Lookup("rootname")
	if ok {
		out = append(out, strings.Split(rootTag, "/"))
	}
	pathTag, ok := f.Tag.Lookup("path")
	if (!ok || pathTag == "") && rootTag == "" {
		return nil, fmt.Errorf("field %s did not specify a path", f.Name)
	}
	if pathTag == "" {
		return out, nil
	}

	ps := strings.Split(pathTag, "|")
	for _, p := range ps {
		sp := removeRootPrefix(strings.Split(p, "/"))
		out = append(out, stripModulePrefixes(sp))
	}
	return out, nil
}

// removeRootPrefix removes the root prefix from root schema entities e.g.
// Bgp_Global has path "/bgp/global" == {"", "bgp", "global"}
//   -> {"global"}
func removeRootPrefix(path []string) []string {
	if len(path) < 2 || path[0] != "" {
		// not a root path
		return path
	}
	return path[2:]
}

// schemaTreeRoot returns the root of the schema tree, given any node in that
// tree. It returns nil if schema is nil.
func schemaTreeRoot(schema *yang.Entry) *yang.Entry {
	if schema == nil {
		return nil
	}

	root := schema
	for root.Parent != nil {
		root = root.Parent
	}

	return root
}

// resolveLeafRef returns a ptr to the schema pointed to by the provided leaf-ref
// schema. It returns schema itself if schema is not a leaf-ref.
func resolveLeafRef(schema *yang.Entry) (*yang.Entry, error) {
	if schema == nil {
		return nil, nil
	}
	if schema.Type == nil {
		// fakeroot
		return schema, nil
	}

	orig := schema
	s := schema
	for ykind := s.Type.Kind; ykind == yang.Yleafref; {
		ns, err := findLeafRefSchema(s)
		if err != nil {
			return schema, err
		}
		s = ns
		ykind = s.Type.Kind
	}

	if s != orig {
		dbgPrintln("follow schema leaf-ref from %s to %s, type %v", orig.Name, s.Name, s.Type.Kind)
	}
	return s, nil
}

// findLeafRefSchema returns the actual pointed to schema if schema is a
// leafref, or schema itself if it is not a leafref.
func findLeafRefSchema(schema *yang.Entry) (*yang.Entry, error) {
	pathStr := schema.Type.Path
	// pathStr has either:
	//  - the relative form "../a/b/../b/c", where ".." indicates the parent of the
	//    node, or
	//  - the absolute form "/a/b/c", which indicates the absolute path from the
	//    root of the schema tree.
	if pathStr == "" {
		return nil, fmt.Errorf("leafref schema %s has empty path", schema.Name)
	}

	refSchema := schema
	pathStr, err := removeXPATHPredicates(pathStr)
	if err != nil {
		return nil, err
	}
	path := strings.Split(pathStr, "/")

	// For absolute path, reset to root of the schema tree.
	if pathStr[0] == '/' {
		refSchema = schemaTreeRoot(schema)
		path = path[1:]
	}

	for i := 0; i < len(path); i++ {
		pe, err := stripPrefix(path[i])
		if err != nil {
			return nil, fmt.Errorf("leafref schema %s path %s: %v", schema.Name, pathStr, err)
		}

		if pe == ".." {
			if refSchema.Parent == nil {
				return nil, fmt.Errorf("parent of %s is nil for leafref schema %s with path %s", refSchema.Name, schema.Name, pathStr)
			}
			refSchema = refSchema.Parent
			continue
		}
		if refSchema.Dir[pe] == nil {
			if isFakeRoot(refSchema) {
				// In the fake root, if we have something at the root of the form /list/container and
				// schema compression is enabled, then we actually have only 'container' at the fake
				// root. So we need to check whether there is a child of the name of the subsequent
				// entry in the path element.
				pech, err := stripPrefix(path[i+1])
				if err != nil {
					return nil, err
				}
				if refSchema.Dir[pech] != nil {
					refSchema = refSchema.Dir[pech]
					// Skip this element.
					i++
					continue
				}
			}
			return nil, fmt.Errorf("schema node %s is nil for leafref schema %s with path %s", pe, schema.Name, pathStr)
		}
		refSchema = refSchema.Dir[pe]
	}

	return refSchema, nil
}

// stripModulePrefixes returns "in" with each element with the format "A:B" changed
// to "B".
func stripModulePrefixes(in []string) []string {
	var out []string
	for _, v := range in {
		out = append(out, stripModulePrefix(v))
	}
	return out
}

// stripModulePrefix returns s with any prefix up to and including the last ':'
// character removed.
func stripModulePrefix(s string) string {
	sv := strings.Split(s, ":")
	return sv[len(sv)-1]
}

// isNil is a general purpose nil check for the kinds of value types expected in
// this package.
func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Map:
		return reflect.ValueOf(value).IsNil()
	}
	return false
}

// isFakeRoot reports whether the supplied yang.Entry represents the synthesised
// root entity in the generated code.
func isFakeRoot(e *yang.Entry) bool {
	if _, ok := e.Annotation["isFakeRoot"]; ok {
		return true
	}
	return false
}

// valueStr returns a string representation of value which may be a value, ptr,
// or struct type.
func valueStr(value interface{}) string {
	kind := reflect.ValueOf(value).Kind()
	switch kind {
	case reflect.Ptr:
		if reflect.ValueOf(value).IsNil() || !reflect.ValueOf(value).IsValid() {
			return "nil"
		}
		return strings.Replace(valueStr(reflect.ValueOf(value).Elem().Interface()), ")", " ptr)", -1)
	case reflect.Struct:
		var out string
		structElems := reflect.ValueOf(value)
		for i := 0; i < structElems.NumField(); i++ {
			if i != 0 {
				out += ", "
			}
			if !structElems.Field(i).CanInterface() {
				continue
			}
			out += valueStr(structElems.Field(i).Interface())
		}
		return "{ " + out + " }"
	}
	out := fmt.Sprintf("%v (type %v)", value, kind)
	if len(out) > maxValueStrLen {
		out = out[:maxValueStrLen] + "..."
	}
	return out
}

// stripPrefix removes the prefix from a YANG path element. For example, removing
// foo from "foo:bar". Such qualified paths are used in YANG modules where remote
// paths are referenced.
func stripPrefix(name string) (string, error) {
	ps := strings.Split(name, ":")
	switch len(ps) {
	case 1:
		return name, nil
	case 2:
		return ps[1], nil
	}
	return "", fmt.Errorf("path element did not form a valid name (name, prefix:name): %v", name)
}

// removeXPATHPredicates removes predicates from an XPath string. e.g.,
// removeXPATHPredicates(/foo/bar[name="foo"]/config/baz -> /foo/bar/config/baz.
func removeXPATHPredicates(s string) (string, error) {
	var b bytes.Buffer
	for i := 0; i < len(s); {
		ss := s[i:]
		si, ei := strings.Index(ss, "["), strings.Index(ss, "]")
		switch {
		case si == -1 && ei == -1:
			// This substring didn't contain a [] pair, therefore write it
			// to the buffer.
			b.WriteString(ss)
			// Move to the last character of the substring.
			i += len(ss)
		case si == -1 || ei == -1:
			// This substring contained a mismatched pair of []s.
			return "", fmt.Errorf("Mismatched brackets within substring %s of %s, [ pos: %d, ] pos: %d", ss, s, si, ei)
		case si > ei:
			// This substring contained a ] before a [.
			return "", fmt.Errorf("Incorrect ordering of [] within substring %s of %s, [ pos: %d, ] pos: %d", ss, s, si, ei)
		default:
			// This substring contained a matched set of []s.
			b.WriteString(ss[0:si])
			i += ei + 1
		}
	}

	return b.String(), nil
}

// getKeyValue returns the value from the structVal field whose last path
// element is key. The value is dereferenced if it is a ptr type. This function
// is used to create a key value for a keyed list.
// getKeyValue returns an error if no path in any of the fields of structVal has
// key as the last path element.
func getKeyValue(structVal reflect.Value, key string) (interface{}, error) {
	for i := 0; i < structVal.NumField(); i++ {
		f := structVal.Type().Field(i)
		p, err := pathToSchema(f)
		if err != nil {
			return nil, err
		}
		if p[len(p)-1] == key {
			fv := structVal.Field(i)
			if fv.Type().Kind() == reflect.Ptr {
				// The type for the key is the dereferenced type, if the type
				// is a ptr.
				if !fv.Elem().IsValid() {
					return nil, fmt.Errorf("key field %s (%s) has nil value %v", key, fv.Type(), fv)
				}
				return fv.Elem().Interface(), nil
			}
			return fv.Interface(), nil
		}
	}

	return nil, fmt.Errorf("could not find key field %s in struct type %s", key, structVal.Type())
}

// toStatus returns a Status with the given code and message.
func toStatus(code scpb.Code, message string) spb.Status {
	return spb.Status{
		Code:    int32(code),
		Message: message,
	}
}

// errToStatus returns a Status with message set to e.Error().
func errToStatus(e error) spb.Status {
	return spb.Status{
		Code:    int32(scpb.Code_INTERNAL),
		Message: e.Error(),
	}
}

// dbgPrint prints v if the package global variable debugLibrary is set.
// v has the same format as Printf.
func dbgPrint(v ...interface{}) {
	if !debugLibrary {
		return
	}
	out := fmt.Sprintf(v[0].(string), v[1:]...)
	if len(out) > maxCharsPerLine {
		out = out[:maxCharsPerLine]
	}
	fmt.Print(globalIndent + out)
}

// dbgPrintNoIndent prints v if the package global variable debugLibrary is set.
// v has the same format as Printf.
func dbgPrintNoIndent(v ...interface{}) {
	tmp := globalIndent
	globalIndent = ""
	dbgPrint(v...)
	globalIndent = tmp
}

// dbgPrintln prints v if the package global variable debugLibrary is set.
// v has the same format as Printf. A trailing newline is added to the output.
func dbgPrintln(v ...interface{}) {
	if !debugLibrary {
		return
	}
	dbgPrint(v...)
	fmt.Println()
}

// globalIndent is used to control indent level.
var globalIndent = ""

// indent increases dbgPrintln indent level.
func indent() {
	globalIndent += ". "
}

// dedent decreases dbgPrintln indent level.
func dedent() {
	globalIndent = strings.TrimPrefix(globalIndent, ". ")
}

// zeroIndent sets the indentation level to zero.
func zeroIndent() {
	globalIndent = ""
}
