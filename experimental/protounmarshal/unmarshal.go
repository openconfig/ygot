package main

import (
	"flag"
	"fmt"
	"reflect"
	"strings"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
	pgen "github.com/golang/protobuf/protoc-gen-go/generator"
	oc "github.com/openconfig/ygot/experimental/protounmarshal/testdata/simple/oc"
	"github.com/openconfig/ygot/ygot"

	spb "github.com/openconfig/ygot/experimental/protounmarshal/testdata/simple/proto"
	ypb "github.com/openconfig/ygot/proto/yext"
	ywpb "github.com/openconfig/ygot/proto/ywrapper"
)

func getProtoFieldPaths(p descriptor.Message) (map[string]map[string]string, error) {
	desc, _ := descriptor.ForMessage(p)

	mm := map[string]map[string]string{}
	for _, m := range desc.MessageType {
		fm := map[string]string{}
		for _, f := range m.Field {
			if f.Name == nil {
				log.Infof("can't interpret field with nil name")
				continue
			}
			var sp string
			if f.Options == nil {
				log.Infof("skipping field %v as options are nil", f.Options)
				continue
			}

			a, err := proto.GetExtension(f.Options, ypb.E_Schemapath)
			if err != nil {
				continue
			}
			sp = *a.(*string)
			fm[sp] = pgen.CamelCase(*f.Name)
			//fm[pgen.CamelCase(*f.Name)] = sp
		}
		mm[*m.Name] = fm
	}
	return mm, nil
}

func goFieldMap(fn map[string][]string) map[string]string {
	mm := map[string]string{}
	for k, v := range fn {
		for _, p := range v {
			mm[p] = k
		}
	}
	return mm
}

func main() {
	flag.Parse()
	x := &spb.A{
		B: &ywpb.StringValue{"test"},
	}

	var protofm map[string]map[string]string
	var err error
	if protofm, err = getProtoFieldPaths(x); err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", protofm)

	y, err := ygot.StructFieldPathMap(&oc.Simple_A{})
	if err != nil {
		log.Infof("%v", err)
	}
	fmt.Printf("%v\n", y)
	gofm := goFieldMap(y)
	fmt.Printf("%v\n", gofm)

	for fp, fv := range protofm["A"] {
		v := reflect.ValueOf(x).Elem().FieldByName(fv)
		if err != nil {
			panic(err)
		}
		pp := strings.Split(fp, "/")
		l := pp[len(pp)-1]

		fmt.Printf("field name: %v\n", l)
		ff, _ := reflect.TypeOf(&oc.Simple_A{}).Elem().FieldByName(gofm[l])
		fmt.Printf("%v\n", ff)
		n := reflect.New(ff.Type)
		n.Elem().Elem().Set(v.Elem().FieldByIndex([]int{0}))
		fmt.Printf("%v %v\n", fp, v)
	}

	fmt.Printf("%v\n", reflect.ValueOf(&oc.Simple_A{B: ygot.String("c")}).Elem().FieldByName("B").Elem().String())
}
