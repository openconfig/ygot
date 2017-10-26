package datatree

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/ygot"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

type TreeNode struct {
	// mu is a a read/write mutex that can be used to acquire a read or
	// write lock on the subtree.
	mu sync.RWMutex
	// subtree is populated if the value in the tree has a subtree itself.
	subtree map[*gnmipb.PathElem]*TreeNode
	// goStruct stores a pointer to the struct that this tree node corresponds
	// with.
	goStruct ygot.GoStruct
	// leaf stores a pointer to a scalar value that this leaf node corresponds
	// to.
	leaf interface{}
}

func (t *TreeNode) IsStruct() bool {
	return t.goStruct != nil
}

func (t *TreeNode) IsLeaf() bool {
	return t.leaf != nil
}

func (t *TreeNode) IsValid() bool {
	if t.IsStruct() && t.IsLeaf() {
		return false
	}
	if t.IsLeaf() && t.subtree != nil {
		return false
	}
	return true
}

func (t *TreeNode) Equal(c *TreeNode) bool {
	if c == nil {
		return false
	}

	c.mu.RLock()
	t.mu.RLock()
	defer t.mu.RUnlock()
	defer c.mu.RUnlock()

	if (t.subtree == nil) != (c.subtree == nil) {
		return false
	}

	if t.subtree != nil {
		if len(t.subtree) != len(c.subtree) {
			return false
		}
	}

	for p, s := range t.subtree {
		k, n := c.find(p)
		if k == nil {
			return false
		}
		if !s.Equal(n) {
			return false
		}
	}

	if !reflect.DeepEqual(t.leaf, c.leaf) {
		return false
	}

	if t.goStruct != c.goStruct {
		return false
	}

	return true
}

// pathForStructField tkaes an input reflect.Value and its corresponding reflect.StructField
// and returns a slice of slices of gnmipb.PathElems that contains the path for the field
// supplied.
func pathForStructField(v reflect.Value, f reflect.StructField) ([][]*gnmipb.PathElem, error) {
	pa, ok := f.Tag.Lookup("path")
	if !ok {
		return nil, fmt.Errorf("field %s did not specify a path", f.Name)
	}

	mapPaths := [][]*gnmipb.PathElem{}
	for _, p := range strings.Split(pa, "|") {
		mp := []*gnmipb.PathElem{}
		for _, pp := range strings.Split(p, "/") {
			e := &gnmipb.PathElem{Name: pp}
			// In the case that the supplied value implements the KeyHelperGoStruct
			// interface, then it is itself a map, so we can use it to get the
			// path of the entity.
			if s, ok := v.Interface().(ygot.KeyHelperGoStruct); ok {
				e.Key = map[string]string{}
				km, err := s.ΛListKeyMap()
				if err != nil {
					return nil, fmt.Errorf("invalid key map for field %s, got: %v", f.Name, err)
				}
				for kn, kv := range km {
					s, err := ygot.KeyValueAsString(kv)
					if err != nil {
						return nil, fmt.Errorf("cannot map key %s to a string: %v", kn, err)
					}
					e.Key[kn] = s
				}
			}
			mp = append(mp, e)
		}
		mapPaths = append(mapPaths, mp)
	}
	return mapPaths, nil
}

func validPathElem(p *gnmipb.PathElem) error {
	if p.Name == "" {
		return fmt.Errorf("nil path element name")
	}

	if _, ok := p.Key[""]; ok {
		return fmt.Errorf("invalid nil value key name")
	}

	return nil
}

func (t *TreeNode) addNode(p *gnmipb.PathElem, c *TreeNode) error {
	if err := validPathElem(p); err != nil {
		return fmt.Errorf("cannot add invalid path element: %v", err)
	}

	if !c.IsValid() {
		return fmt.Errorf("cannot add invalid child at %v", p)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.subtree == nil {
		t.subtree = map[*gnmipb.PathElem]*TreeNode{}
	}

	if k, e := t.find(p); e != nil {
		if e.IsLeaf() != c.IsLeaf() {
			return fmt.Errorf("mismatched types, new isLeaf: %v, existing isLeaf: %v", c.IsLeaf(), e.IsLeaf())
		}
		delete(t.subtree, k)
	}

	t.subtree[p] = c
	return nil
}

// find returns the key pointer, and node corresponding to the path p in the tree node t.
// Since the input pointer may be a pointer to a different PathElem instance, the
// protobuf.Equal is used to compare the input and existing PathElems. Note
// that find *does not* acquire a read lock on the TreeNode - rather the caller
// must ensure that they hold the lock before proceeding if consistency is
// required.
func (t *TreeNode) find(p *gnmipb.PathElem) (*gnmipb.PathElem, *TreeNode) {
	for k := range t.subtree {
		if proto.Equal(k, p) {
			return k, t.subtree[k]
		}
	}
	return nil, nil
}

func (t *TreeNode) addAllNodes(pp []*gnmipb.PathElem, c *TreeNode) error {
	if c == nil || !c.IsValid() {
		return fmt.Errorf("cannot add invalid child at path %v", pp)
	}

	if l := len(pp); l < 2 {
		return fmt.Errorf("invalid length path, got: %d (%v), want: >= 2", l, pp)
	}

	cnode := t
	for i := 0; i < len(pp)-1; i++ {
		if err := validPathElem(pp[i]); err != nil {
			return fmt.Errorf("invalid path element at index %d: %v", i, err)
		}
		if _, n := cnode.find(pp[i]); n != nil {
			if n.IsLeaf() {
				return fmt.Errorf("cannot add branch to %v, is a leaf", pp[i])
			}
			cnode = n
		} else {
			nn := &TreeNode{}
			if err := cnode.addNode(pp[i], nn); err != nil {
				return err
			}
			cnode = nn
		}
	}
	return cnode.addNode(pp[len(pp)-1], c)
}

func (t *TreeNode) addChildrenInternal(s ygot.GoStruct) error {
	sv := reflect.ValueOf(s)
	st := sv.Elem().Type()
	for i := 0; i < st.NumField(); i++ {
		fv := sv.Elem().Field(i)
		ft := st.Field(i)

		if fv.IsNil() {
			continue
		}

		if fv.Kind() == reflect.Map {
			for _, key := range fv.MapKeys() {
				v := fv.MapIndex(key)

				var gs ygot.GoStruct
				var ok bool
				if gs, ok = v.Interface().(ygot.GoStruct); !ok {
					return fmt.Errorf("received map that does not consist of structs, index: %v", key.Interface())
				}

				p, err := pathForStructField(v, ft)
				if err != nil {
					return fmt.Errorf("could not generate path for map field: %v", err)
				}
				c := &TreeNode{
					goStruct: gs,
				}
				c.addChildrenInternal(gs)
				t.add(p, c)
			}
			continue
		}

		p, err := pathForStructField(fv, ft)
		if err != nil {
			return fmt.Errorf("cannot determine path for %v: %v", ft.Name, err)
		}

		var c *TreeNode
		switch fv.Interface().(type) {
		case ygot.GoStruct:
			// This is a struct itself, so we need to create a new Tree for it.
			c = &TreeNode{
				goStruct: s,
			}
			c.addChildrenInternal(fv.Interface().(ygot.GoStruct))
		default:
			c = &TreeNode{
				leaf: fv.Interface(),
			}
		}
		t.add(p, c)

	}
	return nil
}

func (t *TreeNode) add(path [][]*gnmipb.PathElem, child *TreeNode) error {
	for _, pp := range path {
		if len(pp) == 1 {
			if err := t.addNode(pp[0], child); err != nil {
				return err
			}
		} else {
			if err := t.addAllNodes(pp, child); err != nil {
				return err
			}
		}
	}
	return nil
}
