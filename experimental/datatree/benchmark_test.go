package datatree

import (
	"testing"

	"github.com/openconfig/ygot/ygot"
)

type benchmarkStruct struct {
	M map[uint32]*benchmarkStructChild `path:"m"`
}

func (*benchmarkStruct) IsYANGGoStruct() {}

type benchmarkStructChild struct {
	M map[uint32]*benchmarkStructChild `path:"m"`
	S *uint32                          `path:"s"`
}

func (*benchmarkStructChild) IsYANGGoStruct() {}

func (b *benchmarkStructChild) ΛListKeyMap() map[string]interface{} {
	return map[string]interface{}{
		"s": *b.S,
	}
}

func createMapEntries(n int) ygot.GoStruct {
	s := &benchmarkStruct{}
	for i := 0; i < n; i++ {
		s.M[uint32(i)] = &benchmarkStructChild{S: ygot.Uint32(uint32(i))}
	}
	return s
}

func toTree(s ygot.GoStruct) (*TreeNode, error) {
	t := &TreeNode{}
	if err := t.addChildrenInternal(s); err != nil {
		return nil, err
	}
	return t, nil
}

func BenchmarkTree(b *testing.B) {

}
