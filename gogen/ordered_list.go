// Copyright 2023 Google Inc.
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

package gogen

import (
	"bytes"
	"fmt"
)

// generatedOrderedMapStruct contains the necessary information to generate an
// ordered map GoStruct as well as any methods on its parent struct.
type generatedOrderedMapStruct struct {
	// StructName is the name of the ordered map struct.
	StructName string
	// KeyName is the name of the key of the ordered map.
	KeyName string
	// ListTypeName is the type name of the list element that form the
	// values of the ordered map.
	ListTypeName string
	// ListFieldName is the name of the parent field that refers to the
	// ordered list element.
	ListFieldName string
	// Keys of the list that is being generated (length = 1 if the list is
	// single keyed).
	Keys []goStructField
	// ParentStructName is the name of the parent of the ordered map
	// struct.
	ParentStructName string
	// YANGPath is the YANG path of the YANG ordered list.
	YANGPath string
}

// OrderedMapTypeName returns the type name of an ordered map given the type
// name of the ordered map list element.
func OrderedMapTypeName(listElemTypeName string) string {
	return fmt.Sprintf("%s_OrderedMap", listElemTypeName)
}

var (
	goOrderedMapParentMethodsTemplate = mustMakeTemplate("orderedMapParentMethods", `
// AppendNew{{ .ListFieldName }} creates a new entry in the {{ .ListFieldName }}
// ordered map of the {{ .ParentStructName }} struct. The keys of the list are
// populated from the input arguments.
func (s *{{ .ParentStructName }}) AppendNew{{ .ListFieldName }}(
  {{- $length := len .Keys -}}
  {{- range $i, $key := .Keys -}}
	{{ $key.Name }} {{ $key.Type -}}
	{{- if ne (inc $i) $length -}}, {{ end -}}
  {{- end -}}
  ) (*{{ .ListTypeName }}, error) {
	if s.{{ .ListFieldName }} == nil {
		s.{{ .ListFieldName }} = &{{ .StructName }}{}
	}
	return s.{{ .ListFieldName }}.AppendNew(
  {{- $length := len .Keys -}}
  {{- range $i, $key := .Keys -}}
	{{ $key.Name }}
	{{- if ne (inc $i) $length -}}, {{ end -}}
  {{- end -}}
  )
}

// Append{{ .ListFieldName }} appends the supplied {{ .ListTypeName }} struct
// to the list {{ .ListFieldName }} of {{ .ParentStructName }}. If the key value(s)
// specified in the supplied {{ .ListTypeName }} already exist in the list, an
// error is returned.
func (s *{{ .ParentStructName }}) Append{{ .ListFieldName }}(v *{{ .ListTypeName }}) error {
	if s.{{ .ListFieldName }} == nil {
		s.{{ .ListFieldName }} = &{{ .StructName }}{}
	}
	return s.{{ .ListFieldName }}.Append(v)
}

// Get{{ .ListFieldName }} retrieves the value with the specified key from the
// {{ .ListFieldName }} map field of {{ .ParentStructName }}. If the receiver
// is nil, or the specified key is not present in the list, nil is returned
// such that Get* methods may be safely chained.
func (s *{{ .ParentStructName }}) Get{{ .ListFieldName }}(
  {{- $length := len .Keys -}}
  {{- range $i, $key := .Keys -}}
	{{ $key.Name }} {{ $key.Type -}}
	{{- if ne (inc $i) $length -}}, {{ end -}}
  {{- end -}}
  ) *{{ .ListTypeName }} {
	if s == nil {
		return nil
	}
	{{ if gt (len .Keys) 1 -}}
	key := {{ .KeyName }}{
		{{- range $key := .Keys }}
		{{ $key.Name }}: {{ $key.Name }},
		{{- end }}
	}
	{{- else -}}
	{{- range $key := .Keys -}}
	key := {{ $key.Name }}
	{{- end -}}
	{{- end }}
	return s.{{ .ListFieldName }}.Get(key)
}

// Delete{{ .ListFieldName }} deletes the value with the specified keys from
// the receiver {{ .ParentStructName }}. If there is no such element, the
// function is a no-op.
func (s *{{ .ParentStructName }}) Delete{{ .ListFieldName }}(
  {{- $length := len .Keys -}}
  {{- range $i, $key := .Keys -}}
	{{ $key.Name }} {{ $key.Type -}}
	{{- if ne (inc $i) $length -}}, {{ end -}}
  {{- end -}}
  ) bool {
	{{ if gt (len .Keys) 1 -}}
	key := {{ .KeyName }}{
		{{- range $key := .Keys }}
		{{ $key.Name }}: {{ $key.Name }},
		{{- end }}
	}
	{{- else -}}
	{{- range $key := .Keys -}}
	key := {{ $key.Name }}
	{{- end -}}
	{{- end }}
	return s.{{ .ListFieldName }}.Delete(key)
}
`)

	goOrderedMapTemplate = mustMakeTemplate("orderedMap", `
// {{ .StructName }} is an ordered map that represents the "ordered-by user"
// list elements at {{ .YANGPath }}.
type {{ .StructName }} struct {
	keys []{{ .KeyName }}
	valueMap map[{{ .KeyName }}]*{{ .ListTypeName }}
}

// IsYANGOrderedList ensures that {{ .StructName }} implements the
// ygot.GoOrderedMap interface.
func (*{{ .StructName }}) IsYANGOrderedList() {}

// init initializes any uninitialized values.
func (o *{{ .StructName }}) init() {
	if o == nil {
		return
	}
	if o.valueMap == nil {
		o.valueMap = map[{{ .KeyName }}]*{{ .ListTypeName }}{}
	}
}

// Keys returns a copy of the list's keys.
func (o *{{ .StructName }}) Keys() []{{ .KeyName }} {
	if o == nil {
		return nil
	}
	return append([]{{ .KeyName }}{}, o.keys...)
}

// Values returns the current set of the list's values in order.
func (o *{{ .StructName }}) Values() []*{{ .ListTypeName }} {
	if o == nil {
		return nil
	}
	var values []*{{ .ListTypeName }}
	for _, key := range o.keys {
		values = append(values, o.valueMap[key])
	}
	return values
}

// Len returns a size of {{ .StructName }}
func (o *{{ .StructName }}) Len() int {
	if o == nil {
		return 0
	}
	return len(o.keys)
}

// Get returns the value corresponding to the key. If the key is not found, nil
// is returned.
func (o *{{ .StructName }}) Get(key {{ .KeyName }}) *{{ .ListTypeName }} {
	if o == nil {
		return nil
	}
	val, _ := o.valueMap[key]
	return val
}

// Delete deletes an element.
func (o *{{ .StructName }}) Delete(key {{ .KeyName }}) bool {
	if o == nil {
		return false
	}
	if _, ok := o.valueMap[key]; !ok {
		return false
	}
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			delete(o.valueMap, key)
			return true
		}
	}
	return false
}

// Append appends a {{ .ListTypeName }}, returning an error if the key
// already exists in the ordered list or if the key is unspecified.
func (o *{{ .StructName }}) Append(v *{{ .ListTypeName }}) error {
	if o == nil {
		return fmt.Errorf("nil ordered map, cannot append {{ .ListTypeName }}")
	}
	if v == nil {
		return fmt.Errorf("nil {{ .ListTypeName }}")
	}
	{{ if gt (len .Keys) 1 -}}
	{{- range $key := .Keys }}
	{{- if $key.IsScalarField -}}
	if v.{{ $key.Name }} == nil {
		return fmt.Errorf("invalid nil key for {{ $key.Name }}")
	}
	{{ end -}}
	{{- end -}}
	key := {{ .KeyName }}{
		{{- range $key := .Keys }}
		{{- if $key.IsScalarField }}
		{{ $key.Name }}: *v.{{ $key.Name }},
		{{- else }}
		{{ $key.Name }}: v.{{ $key.Name }},
		{{- end -}}
		{{ end }}
	}
	{{- else -}}
	{{- range $key := .Keys -}}
		{{- if $key.IsScalarField -}}
	if v.{{ $key.Name }} == nil {
		return fmt.Errorf("invalid nil key received for {{ $key.Name }}")
	}

	key := *v.{{ $key.Name }}
		{{- else -}}
	key := v.{{ $key.Name }}
		{{- end -}}
	{{- end -}}
	{{- end }}

	if _, ok := o.valueMap[key]; ok {
		return fmt.Errorf("duplicate key for list Statement %v", key)
	}
	o.keys = append(o.keys, key)
	o.init()
	o.valueMap[key] = v
	return nil
}

// AppendNew creates and appends a new {{ .ListTypeName }}, returning the
// newly-initialized v. It returns an error if the v already exists.
func (o *{{ .StructName }}) AppendNew(
  {{- $length := len .Keys -}}
  {{- range $i, $key := .Keys -}}
	{{ $key.Name }} {{ $key.Type -}}
	{{- if ne (inc $i) $length -}}, {{ end -}}
  {{- end -}}
  ) (*{{ .ListTypeName }}, error) {
	if o == nil {
		return nil, fmt.Errorf("nil ordered map, cannot append {{ .ListTypeName }}")
	}
	{{ if gt (len .Keys) 1 -}}
	key := {{ .KeyName }}{
		{{- range $key := .Keys }}
		{{ $key.Name }}: {{ $key.Name }},
		{{- end }}
	}
	{{- else -}}
	{{- range $key := .Keys -}}
	key := {{ $key.Name }}
	{{- end -}}
	{{- end }}

	if _, ok := o.valueMap[key]; ok {
		return nil, fmt.Errorf("duplicate key for list Statement %v", key)
	}
	o.keys = append(o.keys, key)
	newElement := &{{ .ListTypeName }}{
		{{- range $key := .Keys }}
		{{- if $key.IsScalarField }}
		{{ $key.Name }}: &{{ $key.Name }},
		{{- else }}
		{{ $key.Name }}: {{ $key.Name }},
		{{- end -}}
		{{- end }}
	}
	o.init()
	o.valueMap[key] = newElement
	return newElement, nil
}
`)
)

func generateOrderedMapParentMethods(buf *bytes.Buffer, method *generatedOrderedMapStruct) error {
	return goOrderedMapParentMethodsTemplate.Execute(buf, method)
}

func generateOrderedMapStruct(buf *bytes.Buffer, method *generatedOrderedMapStruct) error {
	return goOrderedMapTemplate.Execute(buf, method)
}
