package ygot

import gpb "github.com/openconfig/gnmi/proto/gnmi"

const (
	// PathStructInterfaceName is the name for the interface implemented by all
	// generated path structs.
	PathStructInterfaceName = "PathStruct"
	// PathTypeName is the type name of the common embedded struct
	// containing the path information for a path struct.
	PathTypeName = "NodePath"
)

// PathStruct is an interface that is implemented by any generated path struct
// type; it allows for generic handling of a path struct at any node.
type PathStruct interface {
	parent() PathStruct
	relPath() ([]*gpb.PathElem, []error)
}

// NewNodePath is the constructor for NodePath.
func NewNodePath(relSchemaPath []string, keys map[string]interface{}, p PathStruct) NodePath {
	return NodePath{relSchemaPath: relSchemaPath, keys: keys, p: p}
}

// NodePath is a common embedded type within all path structs. It
// keeps track of the necessary information to create the relative schema path
// as a []*gpb.PathElem during later processing using the Resolve() method,
// thereby delaying any errors being reported until that time.
type NodePath struct {
	relSchemaPath []string
	keys          map[string]interface{}
	p             PathStruct
}
