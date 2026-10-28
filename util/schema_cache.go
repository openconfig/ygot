// Copyright 2026 Google Inc.
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

package util

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/openconfig/goyang/pkg/yang"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

var (
	// goOrderedMapType is the reflect.Type of goOrderedMap used to identify
	// ordered-map struct fields without boxing reflect.Value instances.
	goOrderedMapType = reflect.TypeOf((*goOrderedMap)(nil)).Elem()
	// structDescCache maps a struct reflect.Type to its immutable *StructDesc.
	structDescCache sync.Map
)

// FieldDesc holds pre-parsed reflection and YANG struct-tag metadata for a
// single field of a Go struct. Slice fields on FieldDesc are shared across
// callers and must be treated as read-only.
type FieldDesc struct {
	// Index is the zero-based position of the field within its parent struct.
	Index int
	// Field is the cached reflect.StructField descriptor for the field.
	Field reflect.StructField
	// IsAnnotation reports whether the field carries a ygotAnnotation tag.
	IsAnnotation bool
	// HasPathTag reports whether the field declares a path struct tag.
	HasPathTag bool
	// IsMap reports whether the field's static type is a Go map.
	IsMap bool
	// IsSlice reports whether the field's static type is a Go slice.
	IsSlice bool
	// IsPtr reports whether the field's static type is a pointer.
	IsPtr bool
	// IsNonScalarPtr reports whether the field is a pointer to a struct, map, or slice.
	IsNonScalarPtr bool
	// IsOrderedMap reports whether the field's static type implements goOrderedMap.
	IsOrderedMap bool
	// IsInterface reports whether the field's static type is an interface.
	IsInterface bool

	// SchemaPaths contains the module-prefix-stripped paths parsed from the path tag.
	SchemaPaths [][]string
	// SchemaPathsErr is the error produced when parsing SchemaPaths, if any.
	SchemaPathsErr error
	// ShadowSchemaPaths contains the module-prefix-stripped paths parsed from the shadow-path tag.
	ShadowSchemaPaths [][]string
	// AnnotationLeafPaths contains single-element terminal path slices extracted from an annotation field's path tag.
	AnnotationLeafPaths [][]string
	// RelPath is the relative schema path parsed from the path tag.
	RelPath []string
	// RelPathErr is the error produced when parsing RelPath, if any.
	RelPathErr error
	// RelShadowPath is the relative schema path preferring the shadow-path tag when present.
	RelShadowPath []string
	// RelShadowPathErr is the error produced when parsing RelShadowPath, if any.
	RelShadowPathErr error
}

// InitValue initialises fv on an addressable struct when fv is a nil map or an
// eligible nil pointer (all nil pointers when initialiseLeaves is true, or
// non-scalar pointers when initialiseLeaves is false).
func (fd *FieldDesc) InitValue(fv reflect.Value, initialiseLeaves bool) {
	switch {
	case fd.IsPtr && fv.IsNil():
		if initialiseLeaves || fd.IsNonScalarPtr {
			fv.Set(reflect.New(fd.Field.Type.Elem()))
		}
	case fd.IsMap && fv.IsNil():
		fv.Set(reflect.MakeMap(fd.Field.Type))
	}
}

// StructDesc caches reflection and struct-tag metadata for a Go struct type,
// together with per-schema resolved child entries and O(1) path indices.
type StructDesc struct {
	// StructType is the Go struct reflect.Type described by this descriptor.
	StructType reflect.Type
	// Fields contains the per-field metadata in struct declaration order.
	Fields []FieldDesc
	// FirstRelPathErrIdx is the lowest field index with a non-nil RelPathErr, or len(Fields) if none.
	FirstRelPathErrIdx int
	// FirstRelShadowPathErrIdx is the lowest field index with a non-nil RelShadowPathErr, or len(Fields) if none.
	FirstRelShadowPathErrIdx int

	// fieldsByName maps each direct top-level field name to its *FieldDesc.
	fieldsByName map[string]*FieldDesc
	// hasAnonymous reports whether StructType declares any anonymous embedded fields.
	hasAnonymous bool
	// hasEmptyPrefix reports whether any field path trims trailing empty segments to length zero.
	hasEmptyPrefix bool
	// byFirstElem maps a stripped first path element to matching field indices in ascending order.
	byFirstElem map[string][]int
	// bySingleElem maps a stripped single-segment path element to matching field indices in ascending order.
	bySingleElem map[string][]int
	// byElemPair maps stripped first and second path elements to matching field indices in ascending order.
	byElemPair map[[2]string][]int
	// byFirstShadowElem maps a stripped first shadow-path element to matching field indices in ascending order.
	byFirstShadowElem map[string][]int
	// byShadowSingleElem maps a stripped single-segment shadow-path element to matching field indices in ascending order.
	byShadowSingleElem map[string][]int
	// byShadowElemPair maps stripped first and second shadow-path elements to matching field indices in ascending order.
	byShadowElemPair map[[2]string][]int
	// byLastRelElem maps a stripped terminal RelPath element to matching field indices in ascending order.
	byLastRelElem map[string][]int
	// byLastRelShadowElem maps a stripped terminal RelShadowPath element to matching field indices in ascending order.
	byLastRelShadowElem map[string][]int

	// schemas maps a canonical *yang.Entry to its resolved *StructSchemaDesc.
	schemas sync.Map
	// listContainerSchemas maps a canonical list *yang.Entry to a *StructSchemaDesc resolved with ListAttr cleared.
	listContainerSchemas sync.Map
}

// DescribeStructType returns the cached *StructDesc for struct type t,
// computing and storing it on first access. It panics if t.Kind() is not
// reflect.Struct.
func DescribeStructType(t reflect.Type) *StructDesc {
	if v, ok := structDescCache.Load(t); ok {
		return v.(*StructDesc)
	}
	sd := newStructDesc(t)
	actual, _ := structDescCache.LoadOrStore(t, sd)
	return actual.(*StructDesc)
}

// HasEmptyPrefix reports whether any field path on sd trims trailing empty
// segments to length zero, requiring callers to fall back to a full field scan.
func (sd *StructDesc) HasEmptyPrefix() bool {
	return sd.hasEmptyPrefix
}

// CandidateCursor yields unique candidate struct field indices in ascending
// declaration order across up to four pre-sorted index slices without heap
// allocation.
type CandidateCursor struct {
	// lists holds the remaining sorted candidate field-index slices to merge.
	lists [4][]int
}

// Next returns the next unique field index strictly less than limit and true,
// or 0 and false when no candidate below limit remains.
func (c *CandidateCursor) Next(limit int) (int, bool) {
	best := limit
	found := false
	for i := range c.lists {
		if len(c.lists[i]) > 0 && c.lists[i][0] < best {
			best = c.lists[i][0]
			found = true
		}
	}
	if !found {
		return 0, false
	}
	for i := range c.lists {
		if len(c.lists[i]) > 0 && c.lists[i][0] == best {
			c.lists[i] = c.lists[i][1:]
		}
	}
	return best, true
}

// Candidates returns a stack-allocated CandidateCursor over the field indices
// on sd that can match non-empty gNMI path, optionally including shadow-path
// indices when includeShadow is true.
func (sd *StructDesc) Candidates(path *gpb.Path, includeShadow bool) CandidateCursor {
	elems := path.GetElem()
	if len(elems) == 0 {
		return CandidateCursor{}
	}
	e0 := StripModulePrefix(elems[0].GetName())
	if len(elems) == 1 {
		var c CandidateCursor
		c.lists[0] = sd.byFirstElem[e0]
		if includeShadow {
			c.lists[1] = sd.byFirstShadowElem[e0]
		}
		return c
	}
	key := [2]string{e0, StripModulePrefix(elems[1].GetName())}
	var c CandidateCursor
	c.lists[0] = sd.bySingleElem[e0]
	c.lists[1] = sd.byElemPair[key]
	if includeShadow {
		c.lists[2] = sd.byShadowSingleElem[e0]
		c.lists[3] = sd.byShadowElemPair[key]
	}
	return c
}

// FirstElemFields returns the ascending slice of field indices on sd whose
// SchemaPaths have elem as their module-prefix-stripped first segment. The
// returned slice must not be mutated.
func (sd *StructDesc) FirstElemFields(elem string) []int {
	return sd.byFirstElem[elem]
}

// FirstShadowElemFields returns the ascending slice of field indices on sd
// whose ShadowSchemaPaths have elem as their module-prefix-stripped first
// segment. The returned slice must not be mutated.
func (sd *StructDesc) FirstShadowElemFields(elem string) []int {
	return sd.byFirstShadowElem[elem]
}

// LastRelElemFields returns the ascending slice of field indices on sd whose
// relative schema path (preferring shadow-path when preferShadowPath is true)
// ends with the module-prefix-stripped segment elem. The returned slice must
// not be mutated.
func (sd *StructDesc) LastRelElemFields(elem string, preferShadowPath bool) []int {
	if preferShadowPath {
		return sd.byLastRelShadowElem[elem]
	}
	return sd.byLastRelElem[elem]
}

// FieldByName returns the StructField named name on sd in O(1) for direct
// fields, falling back to reflect.Type.FieldByName when sd embeds anonymous
// structs.
func (sd *StructDesc) FieldByName(name string) (reflect.StructField, bool) {
	if fd, ok := sd.fieldsByName[name]; ok {
		return fd.Field, true
	}
	if sd.hasAnonymous {
		return sd.StructType.FieldByName(name)
	}
	return reflect.StructField{}, false
}

// FieldValueByName returns the StructField and corresponding reflect.Value for
// the field named name on struct value sv.
func (sd *StructDesc) FieldValueByName(sv reflect.Value, name string) (reflect.StructField, reflect.Value, bool) {
	if fd, ok := sd.fieldsByName[name]; ok {
		return fd.Field, sv.Field(fd.Index), true
	}
	if sd.hasAnonymous {
		if ft, ok := sd.StructType.FieldByName(name); ok {
			return ft, sv.FieldByIndex(ft.Index), true
		}
	}
	return reflect.StructField{}, reflect.Value{}, false
}

// Schema returns the cached *StructSchemaDesc resolving all fields of sd
// against schema, computing and storing it on first access.
func (sd *StructDesc) Schema(schema *yang.Entry) *StructSchemaDesc {
	if v, ok := sd.schemas.Load(schema); ok {
		return v.(*StructSchemaDesc)
	}
	targetMap, key := sd.canonicalSchemaKey(schema)
	if v, ok := targetMap.Load(key); ok {
		return v.(*StructSchemaDesc)
	}
	ssd := newStructSchemaDesc(sd, schema)
	actual, _ := targetMap.LoadOrStore(key, ssd)
	return actual.(*StructSchemaDesc)
}

// canonicalSchemaKey resolves shallow copies of list or container *yang.Entry
// values (such as copies created with ListAttr cleared during list element
// traversal) back to their canonical parent-directory entry so ephemeral
// pointers do not accumulate in sync.Map caches.
func (sd *StructDesc) canonicalSchemaKey(schema *yang.Entry) (*sync.Map, *yang.Entry) {
	if schema == nil || schema.Parent == nil {
		return &sd.schemas, schema
	}
	canon := schema.Parent.Dir[schema.Name]
	if canon == nil || canon == schema || !mapsShareEntries(canon.Dir, schema.Dir) {
		return &sd.schemas, schema
	}
	switch {
	case canon.IsContainer() == schema.IsContainer():
		return &sd.schemas, canon
	case canon.IsList() && schema.IsContainer():
		return &sd.listContainerSchemas, canon
	default:
		return &sd.schemas, schema
	}
}

// StructSchemaDesc caches resolved child *yang.Entry values and schema-lookup
// errors for every field of a StructDesc against a specific parent *yang.Entry.
type StructSchemaDesc struct {
	// desc is the parent struct descriptor whose fields correspond to the slices below.
	desc *StructDesc
	// ChildSchemas holds the resolved child schema per field index using path tags.
	ChildSchemas []*yang.Entry
	// ChildSchemaErrs holds the child schema resolution error per field index using path tags.
	ChildSchemaErrs []error
	// ShadowChildSchemas holds the resolved child schema per field index preferring shadow-path tags.
	ShadowChildSchemas []*yang.Entry
	// ShadowChildSchemaErrs holds the child schema resolution error per field index preferring shadow-path tags.
	ShadowChildSchemaErrs []error
	// FirstErrIdx is the lowest field index that unconditionally fails schema or path resolution when preferShadowPath is false, or len(Fields) if none.
	FirstErrIdx int
	// FirstShadowErrIdx is the lowest field index that fails schema resolution or lacks a path tag when preferShadowPath is true, or len(Fields) if none.
	FirstShadowErrIdx int
	// FirstNonAnnotErrIdx is the lowest non-annotation field index whose ChildSchemas entry is nil or errored, or len(Fields) if none.
	FirstNonAnnotErrIdx int
}

// ChildSchema returns the cached child schema and error for field index idx on
// ssd, delegating to childSchema when debugSchema logging is enabled so debug
// output remains unchanged.
func (ssd *StructSchemaDesc) ChildSchema(schema *yang.Entry, idx int, preferShadowPath bool) (*yang.Entry, error) {
	if debugSchema {
		return childSchema(schema, ssd.desc.Fields[idx].Field, preferShadowPath)
	}
	if preferShadowPath {
		return ssd.ShadowChildSchemas[idx], ssd.ShadowChildSchemaErrs[idx]
	}
	return ssd.ChildSchemas[idx], ssd.ChildSchemaErrs[idx]
}

// newStructDesc constructs an immutable *StructDesc for struct type t.
func newStructDesc(t reflect.Type) *StructDesc {
	n := t.NumField()
	sd := &StructDesc{
		StructType:               t,
		Fields:                   make([]FieldDesc, n),
		FirstRelPathErrIdx:       n,
		FirstRelShadowPathErrIdx: n,
		fieldsByName:             make(map[string]*FieldDesc, n),
		byFirstElem:              map[string][]int{},
		bySingleElem:             map[string][]int{},
		byElemPair:               map[[2]string][]int{},
		byFirstShadowElem:        map[string][]int{},
		byShadowSingleElem:       map[string][]int{},
		byShadowElemPair:         map[[2]string][]int{},
		byLastRelElem:            map[string][]int{},
		byLastRelShadowElem:      map[string][]int{},
	}

	for i := 0; i < n; i++ {
		ft := t.Field(i)
		kind := ft.Type.Kind()
		isPtr := kind == reflect.Ptr
		var isNonScalarPtr bool
		if isPtr {
			elemKind := ft.Type.Elem().Kind()
			isNonScalarPtr = elemKind == reflect.Struct || elemKind == reflect.Map || elemKind == reflect.Slice
		}

		isAnnot := IsYgotAnnotation(ft)
		pathTag, hasPathTag := ft.Tag.Lookup("path")

		var annotLeafPaths [][]string
		if isAnnot && hasPathTag {
			annotLeafPaths = parseAnnotationLeafPaths(pathTag)
		}

		sp, spErr := SchemaPaths(ft)
		ssp := ShadowSchemaPaths(ft)
		relPath, relPathErr := relativeSchemaPath(ft, false)
		relShadowPath, relShadowPathErr := relativeSchemaPath(ft, true)

		fd := FieldDesc{
			Index:               i,
			Field:               ft,
			IsAnnotation:        isAnnot,
			HasPathTag:          hasPathTag,
			IsMap:               kind == reflect.Map,
			IsSlice:             kind == reflect.Slice,
			IsPtr:               isPtr,
			IsNonScalarPtr:      isNonScalarPtr,
			IsOrderedMap:        ft.Type.Implements(goOrderedMapType),
			IsInterface:         kind == reflect.Interface,
			SchemaPaths:         sp,
			SchemaPathsErr:      spErr,
			ShadowSchemaPaths:   ssp,
			AnnotationLeafPaths: annotLeafPaths,
			RelPath:             relPath,
			RelPathErr:          relPathErr,
			RelShadowPath:       relShadowPath,
			RelShadowPathErr:    relShadowPathErr,
		}
		sd.Fields[i] = fd

		if ft.Anonymous {
			sd.hasAnonymous = true
		}
		if _, exists := sd.fieldsByName[ft.Name]; !exists {
			sd.fieldsByName[ft.Name] = &sd.Fields[i]
		}

		if relPathErr != nil && i < sd.FirstRelPathErrIdx {
			sd.FirstRelPathErrIdx = i
		} else if relPathErr == nil && len(relPath) > 0 {
			last := StripModulePrefix(relPath[len(relPath)-1])
			sd.byLastRelElem[last] = appendUniqueIndex(sd.byLastRelElem[last], i)
		}

		if relShadowPathErr != nil && i < sd.FirstRelShadowPathErrIdx {
			sd.FirstRelShadowPathErrIdx = i
		} else if relShadowPathErr == nil && len(relShadowPath) > 0 {
			last := StripModulePrefix(relShadowPath[len(relShadowPath)-1])
			sd.byLastRelShadowElem[last] = appendUniqueIndex(sd.byLastRelShadowElem[last], i)
		}

		for _, p := range sp {
			trimmed := trimTrailingEmpty(p)
			if len(trimmed) == 0 {
				sd.hasEmptyPrefix = true
				continue
			}
			first := StripModulePrefix(trimmed[0])
			sd.byFirstElem[first] = appendUniqueIndex(sd.byFirstElem[first], i)
			if len(trimmed) == 1 {
				sd.bySingleElem[first] = appendUniqueIndex(sd.bySingleElem[first], i)
			} else {
				pair := [2]string{first, StripModulePrefix(trimmed[1])}
				sd.byElemPair[pair] = appendUniqueIndex(sd.byElemPair[pair], i)
			}
		}
		for _, p := range ssp {
			trimmed := trimTrailingEmpty(p)
			if len(trimmed) == 0 {
				sd.hasEmptyPrefix = true
				continue
			}
			first := StripModulePrefix(trimmed[0])
			sd.byFirstShadowElem[first] = appendUniqueIndex(sd.byFirstShadowElem[first], i)
			if len(trimmed) == 1 {
				sd.byShadowSingleElem[first] = appendUniqueIndex(sd.byShadowSingleElem[first], i)
			} else {
				pair := [2]string{first, StripModulePrefix(trimmed[1])}
				sd.byShadowElemPair[pair] = appendUniqueIndex(sd.byShadowElemPair[pair], i)
			}
		}
	}

	return sd
}

// newStructSchemaDesc resolves child schemas and first-error indices for every
// field of sd against schema, building the choice/case fallback map at most
// once across all fields.
func newStructSchemaDesc(sd *StructDesc, schema *yang.Entry) *StructSchemaDesc {
	n := len(sd.Fields)
	ssd := &StructSchemaDesc{
		desc:                  sd,
		ChildSchemas:          make([]*yang.Entry, n),
		ChildSchemaErrs:       make([]error, n),
		ShadowChildSchemas:    make([]*yang.Entry, n),
		ShadowChildSchemaErrs: make([]error, n),
		FirstErrIdx:           n,
		FirstShadowErrIdx:     n,
		FirstNonAnnotErrIdx:   n,
	}

	var choiceEntries map[string]*yang.Entry
	var choiceBuilt bool
	lookupChoice := func(seg string) *yang.Entry {
		if !choiceBuilt {
			choiceEntries = buildNonChoiceByStrippedName(schema)
			choiceBuilt = true
		}
		return choiceEntries[seg]
	}

	for i := range sd.Fields {
		fd := &sd.Fields[i]
		if !fd.IsAnnotation {
			if fd.RelPathErr != nil {
				ssd.ChildSchemaErrs[i] = fd.RelPathErr
			} else {
				ssd.ChildSchemas[i] = resolvePathInSchema(schema, fd.RelPath, lookupChoice)
			}

			if fd.RelShadowPathErr != nil {
				ssd.ShadowChildSchemaErrs[i] = fd.RelShadowPathErr
			} else {
				ssd.ShadowChildSchemas[i] = resolvePathInSchema(schema, fd.RelShadowPath, lookupChoice)
			}

			if (ssd.ChildSchemaErrs[i] != nil || ssd.ChildSchemas[i] == nil) && i < ssd.FirstNonAnnotErrIdx {
				ssd.FirstNonAnnotErrIdx = i
			}
		}

		// Record the lowest field index where retrieveNodeContainer fails
		// unconditionally when preferShadowPath is false.
		if ((!fd.IsAnnotation && (ssd.ChildSchemaErrs[i] != nil || ssd.ChildSchemas[i] == nil)) ||
			(fd.IsAnnotation && fd.SchemaPathsErr != nil)) && i < ssd.FirstErrIdx {
			ssd.FirstErrIdx = i
		}

		// Record the lowest field index where retrieveNodeContainer either fails
		// child-schema resolution in Step 1 or lacks a path tag in Step 3 when
		// preferShadowPath is true.
		if ((!fd.IsAnnotation && (ssd.ShadowChildSchemaErrs[i] != nil || ssd.ShadowChildSchemas[i] == nil)) ||
			fd.SchemaPathsErr != nil) && i < ssd.FirstShadowErrIdx {
			ssd.FirstShadowErrIdx = i
		}
	}

	return ssd
}

// resolvePathInSchema resolves a pre-parsed relative schema path p against
// schema, invoking lookupChoice only when the final path element is absent
// from the direct Dir map and eligible for choice/case matching.
func resolvePathInSchema(schema *yang.Entry, p []string, lookupChoice func(string) *yang.Entry) *yang.Entry {
	if schema == nil {
		return nil
	}
	if schema.IsContainer() && len(p) > 1 && p[0] == schema.Name {
		p = p[1:]
	}
	cs := schema
	for ; len(p) > 0; p = p[1:] {
		ns, ok := cs.Dir[StripModulePrefix(p[0])]
		if !ok {
			if len(p) != 1 {
				return nil
			}
			return lookupChoice(p[0])
		}
		cs = ns
	}
	return cs
}

// resolveChildSchemaFromTag resolves the child *yang.Entry for struct field f
// under schema without allocating intermediate path slices or choice maps.
func resolveChildSchemaFromTag(schema *yang.Entry, f reflect.StructField, preferShadowPath bool) (*yang.Entry, error) {
	if IsYgotAnnotation(f) {
		return nil, nil
	}
	selected, err := selectRelativeSchemaPathStr(f, preferShadowPath)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, nil
	}

	if slash := strings.IndexByte(selected, '/'); schema.IsContainer() && slash != -1 && selected[:slash] == schema.Name {
		selected = selected[slash+1:]
	}

	cs := schema
	rem := selected
	for {
		slash := strings.IndexByte(rem, '/')
		var seg string
		if slash == -1 {
			seg = rem
		} else {
			seg = rem[:slash]
		}
		ns, ok := cs.Dir[StripModulePrefix(seg)]
		if !ok {
			if slash != -1 {
				return nil, nil
			}
			return findChoiceCaseChild(schema, seg), nil
		}
		cs = ns
		if slash == -1 {
			return cs, nil
		}
		rem = rem[slash+1:]
	}
}

// selectRelativeSchemaPathStr extracts the raw slash-separated relative schema
// path substring (with any leading slash removed) from f's struct tags without
// allocating intermediate slices. It returns an error if the required tag is
// absent, empty, or contains pipe alternatives with no multi-element path.
func selectRelativeSchemaPathStr(f reflect.StructField, preferShadowPath bool) (string, error) {
	var pathTag string
	var ok bool
	if preferShadowPath {
		pathTag, ok = f.Tag.Lookup("shadow-path")
	}
	switch {
	case !ok:
		if pathTag, ok = f.Tag.Lookup("path"); !ok || pathTag == "" {
			return "", fmt.Errorf("field %s did not specify a path", f.Name)
		}
	case pathTag == "":
		return "", fmt.Errorf("field %s did not specify a shadow-path", f.Name)
	}

	if strings.IndexByte(pathTag, '|') == -1 {
		return strings.TrimPrefix(pathTag, "/"), nil
	}
	rem := pathTag
	for {
		bar := strings.IndexByte(rem, '|')
		var pv string
		if bar == -1 {
			pv = rem
		} else {
			pv = rem[:bar]
		}
		trimmed := strings.TrimPrefix(pv, "/")
		if strings.IndexByte(trimmed, '/') != -1 {
			return trimmed, nil
		}
		if bar == -1 {
			break
		}
		rem = rem[bar+1:]
	}
	return "", fmt.Errorf("field %s had path tag %s with |, but no elements of form a/b", f.Name, pathTag)
}

// findChoiceCaseChild searches the first non-choice, non-case descendants of e
// for an entry whose last path segment (with any module prefix stripped)
// equals target.
func findChoiceCaseChild(e *yang.Entry, target string) *yang.Entry {
	if e == nil {
		return nil
	}
	for _, ch := range e.Dir {
		switch {
		case !IsChoiceOrCase(ch):
			if StripModulePrefix(lastPathElem(ch.Name)) == target {
				return ch
			}
		case ch.IsDir():
			if found := findChoiceCaseChild(ch, target); found != nil {
				return found
			}
		}
	}
	return nil
}

// buildNonChoiceByStrippedName traverses the first non-choice, non-case
// descendants of e once and indexes them by their module-prefix-stripped
// terminal segment name.
func buildNonChoiceByStrippedName(e *yang.Entry) map[string]*yang.Entry {
	if e == nil || len(e.Dir) == 0 {
		return nil
	}
	out := make(map[string]*yang.Entry, len(e.Dir))
	populateNonChoiceByStrippedName(e, out)
	return out
}

// populateNonChoiceByStrippedName recursively populates dst with the first
// non-choice, non-case descendants beneath e.
func populateNonChoiceByStrippedName(e *yang.Entry, dst map[string]*yang.Entry) {
	for _, ch := range e.Dir {
		switch {
		case !IsChoiceOrCase(ch):
			dst[StripModulePrefix(lastPathElem(ch.Name))] = ch
		case ch.IsDir():
			populateNonChoiceByStrippedName(ch, dst)
		}
	}
}

// parseAnnotationLeafPaths parses a pipe-separated annotation path tag into
// single-element slices holding the terminal slash-delimited name of each path.
func parseAnnotationLeafPaths(pathTag string) [][]string {
	n := strings.Count(pathTag, "|") + 1
	elems := make([]string, n)
	out := make([][]string, n)
	rem := pathTag
	for i := 0; i < n; i++ {
		bar := strings.IndexByte(rem, '|')
		var seg string
		if bar == -1 {
			seg = rem
		} else {
			seg = rem[:bar]
			rem = rem[bar+1:]
		}
		elems[i] = lastPathElem(seg)
		out[i] = elems[i : i+1 : i+1]
	}
	return out
}

// lastPathElem returns the substring of s following its last '/' byte, or s
// unchanged if s contains no '/'.
func lastPathElem(s string) string {
	if idx := strings.LastIndexByte(s, '/'); idx != -1 {
		return s[idx+1:]
	}
	return s
}

// trimTrailingEmpty returns p with any trailing empty-string elements removed.
func trimTrailingEmpty(p []string) []string {
	for len(p) > 0 && p[len(p)-1] == "" {
		p = p[:len(p)-1]
	}
	return p
}

// appendUniqueIndex appends idx to indices when indices is empty or its last
// element is not already idx.
func appendUniqueIndex(indices []int, idx int) []int {
	if len(indices) > 0 && indices[len(indices)-1] == idx {
		return indices
	}
	return append(indices, idx)
}

// mapsShareEntries reports whether a and b have identical lengths and map a
// sampled key from b to the same *yang.Entry pointer in a.
func mapsShareEntries(a, b map[string]*yang.Entry) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range b {
		return a[k] == v
	}
	return true
}
