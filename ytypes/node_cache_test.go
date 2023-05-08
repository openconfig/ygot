package ytypes

import (
	"testing"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
)

func TestNodeCacheSizeAndReset(t *testing.T) {
	nodeCache := NewNodeCache()
	inSchema := &Schema{
		Root: &ListElemStruct1{},
		SchemaTree: map[string]*yang.Entry{
			"ListElemStruct1": simpleSchema(),
		},
	}

	err := SetNode(inSchema.RootSchema(), inSchema.Root, mustPath("/outer/inner"), &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
		JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
	}}, &InitMissingElements{}, &NodeCacheOpt{NodeCache: nodeCache})

	if err != nil {
		t.Fatalf("node cache set error: %s", err)
	}

	if nodeCache.Size() != 1 {
		t.Fatalf("expected node cache size 1, got %d", nodeCache.Size())
	}

	nodeCache.Reset()

	if nodeCache.Size() != 0 {
		t.Fatalf("expected node cache size 0, got %d", nodeCache.Size())
	}
}
