package datatree

import (
	"fmt"
	"testing"

	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

var (
	noLeaves int = 1e5
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

func (b *benchmarkStructChild) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{
		"s": *b.S,
	}, nil
}

func createMapEntries(n int) ygot.GoStruct {
	s := &benchmarkStruct{M: map[uint32]*benchmarkStructChild{}}
	for i := 0; i < n; i++ {
		s.M[uint32(i)] = &benchmarkStructChild{S: ygot.Uint32(uint32(i))}
	}
	return s
}

func createHierarchicalMaps(n int) ygot.GoStruct {
	s := &benchmarkStruct{M: map[uint32]*benchmarkStructChild{
		0: &benchmarkStructChild{
			M: map[uint32]*benchmarkStructChild{},
			S: ygot.Uint32(0),
		},
	}}
	p := s.M[0]

	var i uint32
	for i = 1; i < uint32(n); i++ {
		p.M = map[uint32]*benchmarkStructChild{
			0: &benchmarkStructChild{
				M: map[uint32]*benchmarkStructChild{},
				S: ygot.Uint32(i),
			},
		}
		p = p.M[0]
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

func BenchmarkToTreeFlatStruct(b *testing.B) {
	e := createMapEntries(noLeaves)
	for n := 0; n < b.N; n++ {
		_, err := toTree(e)
		if err != nil {
			b.FailNow()
		}
	}
}

func BenchmarkToTreeHierarchicalStruct(b *testing.B) {
	e := createHierarchicalMaps(noLeaves)
	for n := 0; n < b.N; n++ {
		_, err := toTree(e)
		if err != nil {
			b.FailNow()
		}
	}
}

func BenchmarkToNotificationsFlatStruct(b *testing.B) {
	e := createMapEntries(noLeaves)
	for n := 0; n < b.N; n++ {
		_, err := ygot.TogNMINotifications(e, 1, ygot.GNMINotificationsConfig{
			UsePathElem: true,
		})
		if err != nil {
			b.FailNow()
		}
	}
}

func BenchmarkToNotificationsHierarchicalStruct(b *testing.B) {
	e := createHierarchicalMaps(noLeaves)
	for n := 0; n < b.N; n++ {
		_, err := ygot.TogNMINotifications(e, 1, ygot.GNMINotificationsConfig{
			UsePathElem: true,
		})
		if err != nil {
			b.FailNow()
		}
	}
}

func BenchmarkToTreeOC(b *testing.B) {
	d := &exampleoc.Device{}
	for i := 0; i < 1000; i++ {
		d.GetOrCreateInterface(fmt.Sprintf("eth%d", i))
	}
	for n := 0; n < b.N; n++ {
		_, err := toTree(d)
		if err != nil {
			b.FailNow()
		}
	}
}
