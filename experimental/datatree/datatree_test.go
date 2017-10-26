package datatree

import (
	"errors"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/ygot/ygot"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

type simpleGoStruct struct {
	Field *string `path:"simple-field"`
}

func (*simpleGoStruct) IsYANGGoStruct() {}

func TestTreeNodePrimitives(t *testing.T) {
	tests := []struct {
		name       string
		inT        *TreeNode
		wantStruct bool
		wantLeaf   bool
		wantValid  bool
	}{{
		name: "valid leaf",
		inT: &TreeNode{
			leaf: "foo",
		},
		wantLeaf:  true,
		wantValid: true,
	}, {
		name: "invalid leaf - with subtree",
		inT: &TreeNode{
			leaf: "foo",
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "test"}: &TreeNode{},
			},
		},
		wantLeaf: true,
	}, {
		name: "invalid - both leaf and subtree",
		inT: &TreeNode{
			leaf:     "foo",
			goStruct: &simpleGoStruct{ygot.String("foo")},
		},
		wantStruct: true,
		wantLeaf:   true,
		wantValid:  false,
	}, {
		name:      "valid, leaf and struct unset",
		inT:       &TreeNode{},
		wantValid: true,
	}}

	for _, tt := range tests {
		if got := tt.inT.IsStruct(); got != tt.wantStruct {
			t.Errorf("%s: (%v).IsStruct(), did not get expected result, got: %v, want: %v", tt.name, tt.inT, got, tt.wantStruct)
		}

		if got := tt.inT.IsLeaf(); got != tt.wantLeaf {
			t.Errorf("%s: (%v).IsLeaf(), did not get expected result, got: %v, want: %v", tt.name, tt.inT, got, tt.wantLeaf)
		}

		if got := tt.inT.IsValid(); got != tt.wantValid {
			t.Errorf("%s: (%v).IsValid(), did not get expected result, got: %v, want: %v", tt.name, tt.inT, got, tt.wantValid)
		}
	}
}

type multiplePathGoStruct struct {
	Field *string `path:"simple-field|config/simple-field"`
}

func (*multiplePathGoStruct) IsYANGGoStruct() {}

type errorGoStruct struct {
	Field *string
}

func (*errorGoStruct) IsYANGGoStruct() {}

type mapGoStruct struct {
	// Note, this isn't actually an expected input since we expect
	// that this mapGoStructChild is within a map, but this input
	// allows a common testing framework.
	MapElem *mapGoStructChild `path:"elem"`
}

func (*mapGoStruct) IsYANGGoStruct() {}

type mapGoStructChild struct {
	KeyOne *string `path:"key-one"`
	KeyTwo *uint32 `path:"key-two"`
}

func (*mapGoStructChild) IsYANGGoStruct() {}
func (s *mapGoStructChild) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key-one": *s.KeyOne,
		"key-two": *s.KeyTwo,
	}, nil
}

type badMapStructErr struct {
	S *mapGoStructChildErr `path:"s"`
}

func (*badMapStructErr) IsYANGGoStruct() {}

type mapGoStructChildErr struct {
	KeyOne *string `path:"key-one"`
}

func (*mapGoStructChildErr) IsYANGGoStruct() {}
func (*mapGoStructChildErr) ΛListKeyMap() (map[string]interface{}, error) {
	return nil, errors.New("new error")
}

type badMapStructNoStr struct {
	S *mapGoStructChildNoStr `path:"s"`
}

func (*badMapStructNoStr) IsYANGGoStruct() {}

type mapGoStructChildNoStr struct {
	KeyOne *string `path:"key-one"`
}

func (*mapGoStructChildNoStr) IsYANGGoStruct() {}
func (*mapGoStructChildNoStr) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{"foo": ygot.String("err")}, nil
}

func TestPathForStructField(t *testing.T) {
	tests := []struct {
		name      string
		inStruct  ygot.GoStruct
		wantPaths [][]*gnmipb.PathElem
		wantErr   string
	}{{
		name:      "simple struct",
		inStruct:  &simpleGoStruct{ygot.String("value")},
		wantPaths: [][]*gnmipb.PathElem{{{Name: "simple-field"}}},
	}, {
		name:     "multiple paths",
		inStruct: &multiplePathGoStruct{ygot.String("value-two")},
		wantPaths: [][]*gnmipb.PathElem{
			{{Name: "simple-field"}},
			{{Name: "config"}, {Name: "simple-field"}},
		},
	}, {
		name:     "no path specified",
		inStruct: &errorGoStruct{ygot.String("value")},
		wantErr:  "field Field did not specify a path",
	}, {
		name: "map go struct",
		inStruct: &mapGoStruct{
			&mapGoStructChild{
				KeyOne: ygot.String("one"),
				KeyTwo: ygot.Uint32(2),
			},
		},
		wantPaths: [][]*gnmipb.PathElem{
			{{Name: "elem", Key: map[string]string{"key-one": "one", "key-two": "2"}}},
		},
	}, {
		name: "bad map struct, error",
		inStruct: &badMapStructErr{
			&mapGoStructChildErr{KeyOne: ygot.String("one")},
		},
		wantErr: "invalid key map for field S, got: new error",
	}, {
		name: "bad map struct, can't string",
		inStruct: &badMapStructNoStr{
			&mapGoStructChildNoStr{KeyOne: ygot.String("one")},
		},
		wantErr: "cannot map key foo to a string: received a union pointer that didn't contain a struct, got: ptr",
	}}

	for _, tt := range tests {
		sv := reflect.ValueOf(tt.inStruct).Elem()
		st := sv.Type()

		got, err := pathForStructField(sv.Field(0), st.Field(0))
		if err != nil && err.Error() != tt.wantErr {
			t.Errorf("%s: pathForStructField(%v, %v): did not get expected error, got: %v, want: %v", tt.name, sv.Field(0), st.Field(0), err, tt.wantErr)
		}

		if tt.wantErr != "" {
			continue
		}

		for _, a := range got {
			var matched bool
			for _, b := range tt.wantPaths {
				pathEq := true
				if len(a) != len(b) {
					continue
				}
				for i := range b {
					if !proto.Equal(a[i], b[i]) {
						pathEq = false
					}
				}
				if pathEq {
					matched = true
				}
			}
			if !matched {
				t.Errorf("%s: pathForStructField(%v, %v): did not get expected path, got: %v, did not have match in: %v", tt.name, sv.Field(0), st.Field(0), a, tt.wantPaths)
			}
		}
	}
}

func TestEqual(t *testing.T) {
	gsOne := &simpleGoStruct{ygot.String("a")}
	gsTwo := &simpleGoStruct{ygot.String("b")}

	tests := []struct {
		name string
		inA  *TreeNode
		inB  *TreeNode
		want bool
	}{{
		name: "not equal, B nil",
		inA:  &TreeNode{},
	}, {
		name: "not equal, mismatched subtrees",
		inA:  &TreeNode{subtree: map[*gnmipb.PathElem]*TreeNode{}},
		inB:  &TreeNode{},
	}, {
		name: "not equal, different length subtrees",
		inA: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "a"},
			},
		},
		inB: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "a"},
				{Name: "b"}: {leaf: "b"},
			},
		},
	}, {
		name: "not equal, different entries in subtrees",
		inA: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "a"},
			},
		},
		inB: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "b"}: {leaf: "b"},
			},
		},
	}, {
		name: "not equal, different leaf content in node",
		inA: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "a"},
			},
		},
		inB: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "not-equal"},
			},
		},
	}, {
		name: "not equal, different goStruct content in node",
		inA:  &TreeNode{goStruct: gsOne},
		inB:  &TreeNode{goStruct: gsTwo},
	}, {
		name: "equal, same goStruct",
		inA:  &TreeNode{goStruct: gsOne},
		inB:  &TreeNode{goStruct: gsOne},
		want: true,
	}, {
		name: "equal, same leaf",
		inA:  &TreeNode{leaf: ygot.String("a")},
		inB:  &TreeNode{leaf: ygot.String("a")},
		want: true,
	}}

	for _, tt := range tests {
		if got := tt.inA.Equal(tt.inB); got != tt.want {
			t.Errorf("%s: (%v).Equal(%v): did not get expected result, got: %v, want: %v", tt.name, tt.inA, tt.inB, got, tt.want)
		}
	}
}

func TestFind(t *testing.T) {
	sgs := &simpleGoStruct{ygot.String("s")}

	tests := []struct {
		name     string
		inTree   *TreeNode
		inPath   *gnmipb.PathElem
		wantKey  *gnmipb.PathElem
		wantNode *TreeNode
	}{{
		name:   "non-existent node",
		inTree: &TreeNode{},
		inPath: &gnmipb.PathElem{Name: "foo"},
	}, {
		name: "existing element, no key",
		inTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "e"}: &TreeNode{leaf: "eval"},
			},
		},
		inPath:   &gnmipb.PathElem{Name: "e"},
		wantKey:  &gnmipb.PathElem{Name: "e"},
		wantNode: &TreeNode{leaf: "eval"},
	}, {
		name: "existing element with gostruct",
		inTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "e"}: &TreeNode{goStruct: sgs},
			},
		},
		inPath:   &gnmipb.PathElem{Name: "e"},
		wantKey:  &gnmipb.PathElem{Name: "e"},
		wantNode: &TreeNode{goStruct: sgs},
	}, {
		name: "existing element with key",
		inTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "n", Key: map[string]string{"k": "v"}}: {leaf: "bar"},
			},
		},
		inPath:   &gnmipb.PathElem{Name: "n", Key: map[string]string{"k": "v"}},
		wantKey:  &gnmipb.PathElem{Name: "n", Key: map[string]string{"k": "v"}},
		wantNode: &TreeNode{leaf: "bar"},
	}, {
		name: "matching name, but mismatched key",
		inTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "bar", Key: map[string]string{"foo": "bar"}}: {leaf: "baz"},
			},
		},
	}}

	for _, tt := range tests {
		gotKey, gotNode := tt.inTree.find(tt.inPath)
		if !proto.Equal(gotKey, tt.wantKey) {
			t.Errorf("%s: (%v).find(%v): did not get expected key, got: %v, want: %v", tt.name, tt.inTree, tt.inPath, gotKey, tt.wantKey)
		}

		if tt.wantNode == nil && gotNode != nil {
			t.Errorf("%s: (%v).find(%v): did not get expected nil node, got: %v, want: %v", tt.name, tt.inTree, tt.inPath, gotNode, tt.wantNode)
		}

		if tt.wantNode == nil {
			continue
		}

		if !tt.wantNode.Equal(gotNode) {
			diff := pretty.Compare(gotNode, tt.wantNode)
			t.Errorf("%s: (%v).find(%v): did not get expected node when running TreeNode.Equal, diff(-got,+want):\n%s", tt.name, tt.inTree, tt.inPath, diff)
		}
	}
}

func TestValidPathElem(t *testing.T) {
	tests := []struct {
		name       string
		inPathElem *gnmipb.PathElem
		want       string
	}{{
		name:       "valid element with name only",
		inPathElem: &gnmipb.PathElem{Name: "foo"},
	}, {
		name:       "valid element with name and key",
		inPathElem: &gnmipb.PathElem{Name: "foo", Key: map[string]string{"bar": "baz"}},
	}, {
		name:       "invalid, nil element name",
		inPathElem: &gnmipb.PathElem{},
		want:       "nil path element name",
	}, {
		name:       "invalid, nil key name",
		inPathElem: &gnmipb.PathElem{Name: "foo", Key: map[string]string{"": "fish"}},
		want:       "invalid nil value key name",
	}}

	for _, tt := range tests {
		got := validPathElem(tt.inPathElem)
		if (got == nil) != (tt.want == "") {
			t.Errorf("%s: validPathElem(%v), did not get expected result, got: %v, want: %v", tt.name, tt.inPathElem, got, tt.want)
		}
	}
}

func TestAddNode(t *testing.T) {
	tests := []struct {
		name       string
		inNode     *TreeNode
		inPathElem *gnmipb.PathElem
		inChild    *TreeNode
		wantTree   *TreeNode
		wantErr    string
	}{{
		name:       "simple addition of path elem",
		inNode:     &TreeNode{},
		inPathElem: &gnmipb.PathElem{Name: "eone"},
		inChild:    &TreeNode{leaf: "value"},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "eone"}: {leaf: "value"},
			},
		},
	}, {
		name:       "simple addition of path elem with keys",
		inNode:     &TreeNode{},
		inPathElem: &gnmipb.PathElem{Name: "eone", Key: map[string]string{"foo": "bar"}},
		inChild:    &TreeNode{goStruct: &simpleGoStruct{ygot.String("value")}},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "eone", Key: map[string]string{"foo": "bar"}}: &TreeNode{goStruct: &simpleGoStruct{ygot.String("value")}},
			},
		},
	}, {
		name: "addition of a path with matching type",
		inNode: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "a"},
			},
		},
		inPathElem: &gnmipb.PathElem{Name: "a"},
		inChild:    &TreeNode{leaf: "b"},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "b"},
			},
		},
	}, {
		name: "addition of a path with mismatched types, existing leaf",
		inNode: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "a"}: {leaf: "a"},
			},
		},
		inPathElem: &gnmipb.PathElem{Name: "a"},
		inChild:    &TreeNode{goStruct: &simpleGoStruct{ygot.String("b")}},
		wantErr:    `mismatched types, new isLeaf: false, existing isLeaf: true`,
	}, {
		name:       "addition of an invalid element",
		inNode:     &TreeNode{},
		inPathElem: &gnmipb.PathElem{Name: "a"},
		inChild: &TreeNode{
			goStruct: &simpleGoStruct{ygot.String("a")},
			leaf:     "a",
		},
		wantErr: `cannot add invalid child at name:"a" `,
	}, {
		name:       "addition of an element with an invalid path",
		inNode:     &TreeNode{},
		inPathElem: &gnmipb.PathElem{},
		inChild: &TreeNode{
			goStruct: &simpleGoStruct{ygot.String("a")},
		},
		wantErr: "cannot add invalid path element: nil path element name",
	}}

	for _, tt := range tests {
		if err := tt.inNode.addNode(tt.inPathElem, tt.inChild); err != nil && err.Error() != tt.wantErr {
			t.Errorf("%s: (*TreeNode).addNode(%s, %v): did not get expected error, got: %v, want: %v", tt.name, proto.MarshalTextString(tt.inPathElem), tt.inChild, err, tt.wantErr)
		}

		if tt.wantErr != "" {
			continue
		}

		if diff := pretty.Compare(tt.inNode, tt.wantTree); diff != "" {
			t.Errorf("%s: (*TreeNode).addNode(%s, %v): did not get expected tree, diff(-got,+want):\n%s", tt.name, proto.MarshalTextString(tt.inPathElem), tt.inChild, diff)
		}
	}
}

func TestAddAllNodes(t *testing.T) {
	sgs := &simpleGoStruct{ygot.String("test")}
	tests := []struct {
		name     string
		inTree   *TreeNode
		inPath   []*gnmipb.PathElem
		inChild  *TreeNode
		wantTree *TreeNode
		wantErr  string
	}{{
		name:    "error with invalid short path",
		inTree:  &TreeNode{},
		inPath:  []*gnmipb.PathElem{},
		inChild: &TreeNode{},
		wantErr: "invalid length path, got: 0 ([]), want: >= 2",
	}, {
		name:    "error with nil input child",
		inTree:  &TreeNode{},
		inPath:  []*gnmipb.PathElem{{Name: "one"}},
		wantErr: `cannot add invalid child at path [name:"one" ]`,
	}, {
		name:    "add two element path with leaf",
		inTree:  &TreeNode{},
		inPath:  []*gnmipb.PathElem{{Name: "foo"}, {Name: "bar"}},
		inChild: &TreeNode{leaf: "test"},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "foo"}: {
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "bar"}: {leaf: "test"},
					},
				},
			},
		},
	}, {
		name:   "add two element path with gostruct",
		inTree: &TreeNode{},
		inPath: []*gnmipb.PathElem{
			{Name: "foo"},
			{Name: "bar", Key: map[string]string{"one": "two"}},
		},
		inChild: &TreeNode{goStruct: sgs},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "foo"}: {
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "bar", Key: map[string]string{"one": "two"}}: {goStruct: sgs},
					},
				},
			},
		},
	}, {
		name:   "add two element path with leaf",
		inTree: &TreeNode{},
		inPath: []*gnmipb.PathElem{
			{Name: "foo"},
			{Name: "bar"},
		},
		inChild: &TreeNode{leaf: "bar"},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "foo"}: {
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "bar"}: {leaf: "bar"},
					},
				},
			},
		},
	}, {
		name: "add to path where element already exists",
		inTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "config"}: {
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "cheese"}: {leaf: "cheddar"},
					},
				},
			},
		},
		inPath: []*gnmipb.PathElem{
			{Name: "config"},
			{Name: "bar"},
		},
		inChild: &TreeNode{leaf: "baz"},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "config"}: {
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "bar"}:    {leaf: "baz"},
						{Name: "cheese"}: {leaf: "cheddar"},
					},
				},
			},
		},
	}, {
		name:   "add with invalid path element at leaf node",
		inTree: &TreeNode{},
		inPath: []*gnmipb.PathElem{
			{Name: "config"},
			{Name: ""},
		},
		inChild: &TreeNode{leaf: "baz"},
		wantErr: "cannot add invalid path element: nil path element name",
	}, {
		name:   "add with invalid path element at branch node",
		inTree: &TreeNode{},
		inPath: []*gnmipb.PathElem{
			{Name: "config"},
			{Name: ""},
			{Name: "baz"},
		},
		inChild: &TreeNode{leaf: "baz"},
		wantErr: "invalid path element at index 1: nil path element name",
	}, {
		name: "add replacing other type error",
		inTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "config"}: {
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "bar"}: {leaf: "baz"},
					},
				},
			},
		},
		inPath: []*gnmipb.PathElem{
			{Name: "config"},
			{Name: "bar"},
		},
		inChild: &TreeNode{goStruct: sgs},
		wantErr: "mismatched types, new isLeaf: false, existing isLeaf: true",
	}, {
		name: "add with existing parent node",
		inTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "config"}: {
					leaf: "bar",
				},
			},
		},
		inPath: []*gnmipb.PathElem{
			{Name: "config"},
			{Name: "baz"},
		},
		inChild: &TreeNode{leaf: "bar"},
		wantErr: `cannot add branch to name:"config" , is a leaf`,
	}}

	for _, tt := range tests {
		err := tt.inTree.addAllNodes(tt.inPath, tt.inChild)
		switch {
		case err != nil && err.Error() != tt.wantErr:
			t.Errorf("%s: (*TreeNode).addAllNodes(%v, %v): did not get expected error, got: %v, want: %v", tt.name, tt.inPath, tt.inChild, err, tt.wantErr)
			continue
		case err == nil && tt.wantErr != "":
			t.Errorf("%s: (*TreeNode).addAllNodes(%v,%v): did not get expected error, got: nil, want: %v", tt.name, tt.inPath, tt.inChild, tt.wantErr)
			continue
		case tt.wantErr != "":
			continue
		}

		if !tt.wantTree.Equal(tt.inTree) {
			diff := pretty.Compare(tt.inTree, tt.wantTree)
			t.Errorf("%s: (*TreeNode).addAllNodes(%v, %v): did not get expected return tree, diff(-got,+want):\n%s", tt.name, tt.inPath, tt.inChild, diff)
		}
	}
}

type nestedGoStruct struct {
	C *nestedGoStruct `path:"c"`
	L *string         `path:"s"`
}

func (*nestedGoStruct) IsYANGGoStruct() {}

type goStructWithMap struct {
	M map[string]*mapGoStructChild `path:"m"`
}

func (*goStructWithMap) IsYANGGoStruct() {}

type goStructWithInvalidMap struct {
	M map[string]string `path:"f"`
}

func (*goStructWithInvalidMap) IsYANGGoStruct() {}

type goStructWithInvalidPathMap struct {
	M map[string]*mapGoStructChildErr `path:"m"`
}

func (*goStructWithInvalidPathMap) IsYANGGoStruct() {}

func TestAddChildrenInternal(t *testing.T) {
	ns := &nestedGoStruct{
		C: &nestedGoStruct{
			L: ygot.String("value"),
		},
	}

	ms := &goStructWithMap{
		M: map[string]*mapGoStructChild{
			"one": {KeyOne: ygot.String("one"), KeyTwo: ygot.Uint32(1)},
			"two": {KeyOne: ygot.String("two"), KeyTwo: ygot.Uint32(2)},
		},
	}

	mps := &multiplePathGoStruct{
		Field: ygot.String("bar"),
	}

	tests := []struct {
		name     string
		inTree   *TreeNode
		inStruct ygot.GoStruct
		wantTree *TreeNode
		wantErr  string
	}{{
		name:     "simple struct addition",
		inTree:   &TreeNode{},
		inStruct: &simpleGoStruct{ygot.String("foo")},
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "simple-field"}: {leaf: ygot.String("foo")},
			},
		},
	}, {
		name:     "multi-part path addition",
		inTree:   &TreeNode{},
		inStruct: mps,
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "simple-field"}: {leaf: mps.Field},
				{Name: "config"}: {
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "simple-field"}: {leaf: mps.Field},
					},
				},
			},
		},
	}, {
		name:     "struct with a field path missing",
		inTree:   &TreeNode{},
		inStruct: &errorGoStruct{ygot.String("bar")},
		wantErr:  "cannot determine path for Field: field Field did not specify a path",
	}, {
		name:     "nested go structs",
		inTree:   &TreeNode{},
		inStruct: ns,
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "c"}: {
					goStruct: ns,
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "s"}: {
							leaf: ns.C.L,
						},
					},
				},
			},
		},
	}, {
		name:     "struct with a map",
		inTree:   &TreeNode{},
		inStruct: ms,
		wantTree: &TreeNode{
			subtree: map[*gnmipb.PathElem]*TreeNode{
				{Name: "m", Key: map[string]string{"key-one": "one", "key-two": "1"}}: {
					goStruct: ms.M["one"],
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "key-one"}: {leaf: ms.M["one"].KeyOne},
						{Name: "key-two"}: {leaf: ms.M["one"].KeyTwo},
					},
				},
				{Name: "m", Key: map[string]string{"key-one": "two", "key-two": "2"}}: {
					goStruct: ms.M["two"],
					subtree: map[*gnmipb.PathElem]*TreeNode{
						{Name: "key-one"}: {leaf: ms.M["two"].KeyOne},
						{Name: "key-two"}: {leaf: ms.M["two"].KeyTwo},
					},
				},
			},
		},
	}, {
		name:   "invalid map type",
		inTree: &TreeNode{},
		inStruct: &goStructWithInvalidMap{
			M: map[string]string{"foo": "bar"},
		},
		wantErr: "received map that does not consist of structs, index: foo",
	}, {
		name:   "invalid map child",
		inTree: &TreeNode{},
		inStruct: &goStructWithInvalidPathMap{
			M: map[string]*mapGoStructChildErr{"bar": {ygot.String("bar")}},
		},
		wantErr: "could not generate path for map field: invalid key map for field M, got: new error",
	}}

	for _, tt := range tests {
		err := tt.inTree.addChildrenInternal(tt.inStruct)
		if err != nil {
			if err.Error() != tt.wantErr {
				t.Errorf("%s: (*TreeNode).addAllChildren(%v): did not get expected error, got: %v, want: %v", tt.name, tt.inStruct, err, tt.wantErr)
			}
			continue
		}

		if !tt.wantTree.Equal(tt.inTree) {
			diff := pretty.Compare(tt.inTree, tt.wantTree)
			t.Errorf("%s: (*TreeNode).addAllChildren(%v): did not get expected tree, diff(-got,+want):\n%s", tt.name, tt.inStruct, diff)
		}
	}
}
