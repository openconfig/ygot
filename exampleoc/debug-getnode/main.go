package main

import (
	"fmt"
	"reflect"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func main() {
	p := &gpb.Path{
		Origin: "openconfig",
		Elem: []*gpb.PathElem{
			{
				Name: "interfaces",
			},
			{
				Name: "interface",
				Key: map[string]string{
					"name": "Ethernet1",
				},
			},
			{
				Name: "state",
			},
			{
				Name: "oper-status",
			},
		},
	}
	s, _, _ := fetchOcStruct(p)
	ts := time.Now().UnixNano()
	results, _ := ygot.TogNMINotifications(s, ts, ygot.GNMINotificationsConfig{UsePathElem: true})
	fmt.Printf("Result is: %v\n", results)
}

func fetchOcStruct(path *gpb.Path) (ygot.ValidatedGoStruct, string, error) {
	ds := &oc.Device{}
	schema := oc.SchemaTree[reflect.TypeOf(ds).Elem().Name()]
	tn, _, err := ytypes.GetOrCreateNode(schema, ds, path)
	if err != nil {
		fmt.Printf("Error is %v\n", err)
	}
	return tn.(ygot.ValidatedGoStruct), "", nil
}
