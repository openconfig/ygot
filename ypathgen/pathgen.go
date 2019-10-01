// Copyright 2019 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ypathgen contains a library to generate gNMI paths from a YANG model.
// The ygen library is used to parse YANG and obtain intermediate and some final
// information. The output always assumes the OpenConfig-specific conventions
// for a compressed schema.
//
// TODO(wenbli): This package is written with only compressed schemas in mind.
// If needed, can write tests, verify, and enhance it to support uncompressed
// ygen structs.
package ypathgen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
)

// Static names used in the generated code.
const (
	// schemaStructPkgAlias is the package alias for the imported ygen-generated file.
	schemaStructPkgAlias string = "oc"
)

var (
	// goChildConstructorTemplate generates the child constructor method
	// for a generated struct by returning an instantiation of the child's
	// path struct object.
	goChildConstructorTemplate = `
// {{ .MethodName }} returns from {{ .Struct.TypeName }} the path struct for its child "{{ .SchemaName }}".
func (n *{{ .Struct.TypeName }}) {{ .MethodName -}} ({{ .KeyParamListStr }}) *{{ .TypeName }} {
	return &{{ .TypeName }}{
		{{ .Struct.PathTypeName }}: ygot.New{{ .Struct.PathTypeName }}(
			[]string{ {{- .RelPathList -}} },
			map[string]interface{}{ {{- .KeyEntriesStr -}} },
			n,
		),
	}
}
`

	// The set of built templates that are to be referenced during code generation.
	goPathTemplates = map[string]*template.Template{
		"childConstructor": makePathTemplate("childConstructor", goChildConstructorTemplate),
	}
)

// makePathTemplate generates a template.Template for a particular named source template
func makePathTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Parse(src))
}

// goPathStructData stores template information needed to generate a struct
// field's type definition.
type goPathStructData struct {
	TypeName     string // TypeName is the type name of the struct being output.
	YANGPath     string // YANGPath is the schema path of the struct being output.
	PathTypeName string // PathTypeName is the type name of the common embedded path struct.
}

// getStructData returns the goPathStructData corresponding to a Directory,
// which is used to store the attributes of the template for which code is
// being generated.
func getStructData(directory *ygen.Directory) goPathStructData {
	return goPathStructData{
		TypeName:     directory.Name,
		YANGPath:     util.SlicePathToString(directory.Path),
		PathTypeName: ygot.PathTypeName,
	}
}

// goPathFieldData stores template information needed to generate a struct
// field's child constructor method.
type goPathFieldData struct {
	MethodName      string           // MethodName is the name of the method that can be called to get to this field.
	SchemaName      string           // SchemaName is the field's original name in the schema.
	TypeName        string           // TypeName is the type name of the returned struct.
	RelPathList     string           // RelPathList is the list of strings that form the relative path from its containing struct.
	Struct          goPathStructData // Struct stores template information for the field's containing struct.
	KeyParamListStr string           // KeyParamListStr is the parameter list of the field's accessor method.
	KeyEntriesStr   string           // KeyEntriesStr is an ordered list of comma-separated ("schemaName": unique camel-case name) for a list's keys.
}

// generateChildConstructor generates and writes to methodBuf a Go method that
// returns an instantiation of the child node's path struct object. It
// takes as input the buffer to store the method, a directory, the field name
// of the directory identifying the child yang.Entry, a directory-level
// unique field name to be used as the generated method's name and the
// incremental type name of of the child path struct, and a map of all
// directories of the whole schema keyed by their schema paths.
func generateChildConstructor(methodBuf *bytes.Buffer, directory *ygen.Directory, directoryFieldName string, goFieldName string, directories map[string]*ygen.Directory) error {
	field, ok := directory.Fields[directoryFieldName]
	if !ok {
		return fmt.Errorf("generateChildConstructor: field %s not found in directory %v", directoryFieldName, directory)
	}
	fieldTypeName, err := getFieldTypeName(directory, directoryFieldName, goFieldName, directories)
	if err != nil {
		return err
	}

	structData := getStructData(directory)
	relPath, err := ygen.FindSchemaPath(directory, directoryFieldName, false)
	if err != nil {
		return err
	}
	fieldData := goPathFieldData{
		MethodName:  goFieldName,
		TypeName:    fieldTypeName,
		SchemaName:  field.Name,
		Struct:      structData,
		RelPathList: "\"" + strings.Join(relPath, "\", \"") + "\"",
	}

	// This is expected to be nil for leaf fields.
	fieldDirectory := directories[field.Path()]

	if field.IsList() {
		if fieldDirectory.ListAttr == nil {
			// keyless lists as a path are not supported by gNMI. Since this library is currently intended to be used for gNMI, just error out.
			return fmt.Errorf("generateChildConstructor: schemas containing keyless lists are unsupported, path: %s", field.Path())
		}
		keyParamListStr, err := makeParamListStr(fieldDirectory.ListAttr)
		if err != nil {
			return err
		}
		fieldData.KeyParamListStr = keyParamListStr
		fieldData.KeyEntriesStr = makeKeyMapStr(fieldDirectory.ListAttr)
	}

	return goPathTemplates["childConstructor"].Execute(methodBuf, fieldData)
}

// getFieldTypeName returns the type name for a field node of a directory -
// handling the case where the field supplied is a leaf or directory. The input
// directories is a map from paths to directory entries, and goFieldName is
// the incremental type name to be used for the case that the directory field
// is a leaf. For non-leaves, their corresponding directory's "Name" is re-used
// as their type names; for leaves, type names are synthesized.
func getFieldTypeName(directory *ygen.Directory, directoryFieldName string, goFieldName string, directories map[string]*ygen.Directory) (string, error) {
	field, ok := directory.Fields[directoryFieldName]
	if !ok {
		return "", fmt.Errorf("getFieldTypeName: field %s not found in directory %v", directoryFieldName, directory)
	}

	if !field.IsLeaf() && !field.IsLeafList() {
		fieldDirectory, ok := directories[field.Path()]
		if !ok {
			return "", fmt.Errorf("getFieldTypeName: unexpected - field %s not found in parsed yang structs map: %v", field.Path(), directories)
		}
		return fieldDirectory.Name, nil
	}

	// Leaves do not have corresponding Directory entries, so their names need to be constructed.
	if isTopLevelLeaf := directory.Entry.Parent == nil; isTopLevelLeaf {
		// When a leaf resides at the root, its type name is its whole name -- we never want fakeroot's name as a prefix.
		return goFieldName, nil
	}
	return directory.Name + "_" + goFieldName, nil
}

// makeKeyMapStr returns a literal instantiation of the map of key name to
// values to be assigned to the "key" attribute of a path node; the enveloping
// map[string]interface{} is omitted from this output. The name of the value
// variable is camel-cased and uniquified, and done in an identical manner as
// makeParamListStr to ensure compilation.
// e.g.
// in: &ygen.YangListAttr{
// 	Keys: map[string]*ygen.MappedType{
// 		"fluorine": &ygen.MappedType{NativeType: "string"},
// 		"iodine-liquid":   &ygen.MappedType{NativeType: "Binary"},
// 	},
// 	KeyElems: []*yang.Entry{{Name: "fluorine"}, {Name: "iodine-liquid"}},
// }
// out: "fluorine": Fluorine, "iodine-liquid": IodineLiquid
func makeKeyMapStr(listAttr *ygen.YangListAttr) string {
	var entries []string
	goKeyNameMap := getGoKeyNameMap(listAttr.KeyElems)
	for _, key := range listAttr.KeyElems { // NOTE: loop on list for deterministic output.
		entries = append(entries, fmt.Sprintf("\"%s\": %s", key.Name, goKeyNameMap[key.Name]))
	}

	return strings.Join(entries, ", ")
}

// makeParamListStr generates the go parameter list for a child list's
// constructor method given the list's ygen.YangListAttr.
// e.g.
// in: &ygen.YangListAttr{
// 	Keys: map[string]*ygen.MappedType{
// 		"fluorine": &ygen.MappedType{NativeType: "string"},
// 		"iodine-liquid":   &ygen.MappedType{NativeType: "Binary"},
// 	},
// 	KeyElems: []*yang.Entry{{Name: "fluorine"}, {Name: "iodine-liquid"}},
// }
// out: "Fluorine string, IodineLiquid oc.Binary"
func makeParamListStr(listAttr *ygen.YangListAttr) (string, error) {
	if len(listAttr.KeyElems) == 0 {
		return "", fmt.Errorf("makeParamListStr: invalid list - has no key; cannot process param list string")
	}

	// Create parameter list *in order* of keys, which should be in schema order.
	var entries []string
	// NOTE: Although the generated key names might not match their
	// corresponding ygen field names in case of a camelcase name
	// collision, we expect that the user is aware of the schema to know
	// the order of the keys, and not rely on the naming in that case.
	goKeyNameMap := getGoKeyNameMap(listAttr.KeyElems)
	for _, keyElem := range listAttr.KeyElems {
		mappedType, ok := listAttr.Keys[keyElem.Name]
		switch {
		case !ok:
			return "", fmt.Errorf("makeParamListStr: key doesn't have a mappedType: %s", keyElem.Name)
		case mappedType == nil:
			return "", fmt.Errorf("makeParamListStr: mappedType for key is nil: %s", keyElem.Name)
		}

		var typeName string
		switch {
		case mappedType.NativeType == "interface{}": // ygen-unsupported types
			typeName = "string"
		case ygen.IsYgenDefinedGoType(mappedType):
			typeName = schemaStructPkgAlias + "." + mappedType.NativeType
		default:
			typeName = mappedType.NativeType
		}

		entries = append(entries, fmt.Sprintf("%s %s", goKeyNameMap[keyElem.Name], typeName))
	}
	return strings.Join(entries, ", "), nil
}

// getGoKeyNameMap returns a map of Go key names keyed by their schema names
// given a list of key entries. Names are camelcased and uniquified to ensure
// compilation. Uniqification is done deterministically.
func getGoKeyNameMap(keyElems []*yang.Entry) map[string]string {
	goKeyNameMap := make(map[string]string, len(keyElems))

	usedKeyNames := map[string]bool{}
	for _, keyElem := range keyElems {
		goKeyNameMap[keyElem.Name] = genutil.MakeNameUnique(genutil.EntryCamelCaseName(keyElem), usedKeyNames)
	}
	return goKeyNameMap
}
