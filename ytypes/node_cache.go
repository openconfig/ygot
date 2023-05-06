package ytypes

import (
	"fmt"
	"strings"
	"sync"

	json "github.com/goccy/go-json"
	"github.com/openconfig/ygot/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// NodeCacheOpt is an option that can be usd with calls such as `SetNode` and `GetNode`.
//
// The `node cache` potentially provides fast-paths for config tree traversals and offers
// noticeable performance boosts on a busy server that calls functions such as `SetNode`
// and `GetNode` frequently.
//
// Passing in the pointer of the `node cache` would prevent making the `ytypes` package
// stateful. The applications that use the `ytypes` package maintain the `node cache`.
type NodeCacheOpt struct {
	NodeCache *NodeCache
}

// IsGetNodeOpt implements the GetNodeOpt interface.
func (*NodeCacheOpt) IsGetNodeOpt() {}

// IsSetNodeOpt implements the SetNodeOpt interface.
func (*NodeCacheOpt) IsSetNodeOpt() {}

// IsUnmarshalOpt marks IgnoreExtraFields as a valid UnmarshalOpt.
func (*NodeCacheOpt) IsUnmarshalOpt() {}

// IsGetOrCreateNodeOpt implements the GetOrCreateNodeOpt interface.
func (*NodeCacheOpt) IsGetOrCreateNodeOpt() {}

// IsDelNodeOpt implements the DelNodeOpt interface.
func (*NodeCacheOpt) IsDelNodeOpt() {}

// cachedNodeInfo is used to provide shortcuts to making operations
// to the nodes without having to traverse the config tree for every operation.
type cachedNodeInfo struct {
	parent interface{}
	root   interface{}
	nodes  []*TreeNode
}

// NodeCache is a thread-safe struct that's used for providing fast-paths for config tree traversals.
type NodeCache struct {
	mu    *sync.RWMutex
	store map[string]*cachedNodeInfo
}

// NewNodeCache returns the pointer of a new `NodeCache` instance.
func NewNodeCache() *NodeCache {
	return &NodeCache{
		mu:    &sync.RWMutex{},
		store: map[string]*cachedNodeInfo{},
	}
}

// Reset resets the cache. The cache will be repopulated after the reset.
func (c *NodeCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = map[string]*cachedNodeInfo{}
}

// setNodeCache uses the cached information to set the node instead of traversalling the config tree.
// This improves runtime performance of the library.
func (c *NodeCache) set(path *gpb.Path, val interface{}, opts ...SetNodeOpt) (setComplete bool, err error) {
	switch val.(type) {
	case *gpb.TypedValue:
	default:
		// Only TypedValue is supported by the node cache for now.
		return
	}

	// Get the unique path representation as the key of the cache store.
	pathRep, err := uniquePathRepresentation(path)
	if err != nil {
		return
	}

	c.mu.Lock()

	// If the path was cached, use the shortcut for setting the node value.
	nodeInfo, ok := c.store[pathRep]
	if !ok || len(nodeInfo.nodes) == 0 {
		c.mu.Unlock()
		return
	}

	schema := &nodeInfo.nodes[0].Schema
	parent := &nodeInfo.parent
	root := &nodeInfo.root

	// Set value in the config tree.
	switch {
	case val.(*gpb.TypedValue).GetJsonIetfVal() != nil:
	default:
		var encoding Encoding
		var options []UnmarshalOpt
		if hasSetNodePreferShadowPath(opts) {
			options = append(options, &PreferShadowPath{})
		}

		if hasIgnoreExtraFieldsSetNode(opts) {
			options = append(options, &IgnoreExtraFields{})
		}

		if hasTolerateJSONInconsistencies(opts) {
			encoding = gNMIEncodingWithJSONTolerance
		} else {
			encoding = GNMIEncoding
		}

		// This call updates the node's value in the config tree.
		if e := unmarshalGeneric(*schema, *parent, val, encoding, options...); e != nil {
			// When the node doesn't exist the unmarshal call will fail. Within the setNodeCache function this
			// should not return an error. Instead, this should return setComplete == false to let the ygot library
			// continue to go through the node creation process.
			fmt.Printf("node cache - unmarshalling error: %s\n", e)
			c.mu.Unlock()
			return
		}
	}

	c.mu.Unlock()

	// Retrieve the node and update the cache.
	var nodes []*TreeNode
	nodes, err = retrieveNode(*schema, *parent, *root, nil, path, retrieveNodeArgs{
		modifyRoot:                        hasInitMissingElements(opts),
		val:                               val,
		tolerateJSONInconsistenciesForVal: hasTolerateJSONInconsistencies(opts),
		preferShadowPath:                  hasSetNodePreferShadowPath(opts),
		ignoreExtraFields:                 hasIgnoreExtraFieldsSetNode(opts),
		uniquePathRepresentation:          &pathRep,
		nodeCache:                         c,
	})
	if err != nil {
		// Here it's assumed that the set was successful. Therefore, if an error is
		// returned from retrieveNode the error should be escalated.
		return
	}

	if len(nodes) != 0 {
		setComplete = true
		return
	}

	err = status.Errorf(
		codes.Unknown,
		"failed to retrieve node, value %v",
		val,
	)

	return
}

// update performs `NodeCache` update based on the input arguments.
func (c *NodeCache) update(nodes []*TreeNode, tp, np *gpb.Path, parent, root interface{}, pathStr *string) {
	var pathRep string
	if pathStr != nil {
		pathRep = *pathStr
	} else {
		if tp != nil && len(tp.GetElem()) > 0 {
			var err error

			pathRep, err = uniquePathRepresentation(appendElem(np, tp.GetElem()[0]))
			if err != nil {
				return
			}
		} else {
			var err error

			pathRep, err = uniquePathRepresentation(np)
			if err != nil {
				return
			}
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.store[pathRep]; !ok {
		c.store[pathRep] = &cachedNodeInfo{}
	}

	if c.store[pathRep].nodes == nil || len(c.store[pathRep].nodes) == 0 {
		c.store[pathRep].nodes = nodes
	} else {
		// Only update the data.
		c.store[pathRep].nodes[0].Data = nodes[0].Data
	}

	c.store[pathRep].parent = parent
	c.store[pathRep].root = root
}

// delete removes the path entry from the node cache.
func (c *NodeCache) delete(path *gpb.Path) {
	// Delete in the nodeCache.
	nodePath, err := uniquePathRepresentation(path)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	keysToDelete := []string{}
	for k := range c.store {
		if strings.Contains(k, nodePath) {
			keysToDelete = append(keysToDelete, k)
		}
	}

	for _, k := range keysToDelete {
		delete(c.store, k)
	}
}

// get tries to retrieve the cached `TreeNode` slice. If the cache doesn't contain the
// target `TreeNode` slice or the cached data is `nil`, an error is returned.
func (c *NodeCache) get(path *gpb.Path) ([]*TreeNode, error) {
	pathRep, err := uniquePathRepresentation(path)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if nodeInfo, ok := c.store[pathRep]; ok {
		ret := nodeInfo.nodes
		if len(ret) == 0 {
			return nil, status.Error(codes.NotFound, "cache: no node was found.")
		} else if util.IsValueNil(ret[0].Data) {
			return nil, status.Error(codes.NotFound, "cache: nil value node was found.")
		}

		return ret, nil
	}

	return nil, nil
}

// uniquePathRepresentation returns the unique path representation. The current
// implementation uses json marshal for both simplicity and performance.
//
// https://pkg.go.dev/encoding/json#Marshal
//
// Map values encode as JSON objects. The map's key type must either be a string,
// an integer type, or implement encoding.TextMarshaler. The map keys are sorted
// and used as JSON object keys by applying the following rules, subject to the
// UTF-8 coercion described for string values above:
//
//   - keys of any string type are used directly
//
//   - encoding.TextMarshalers are marshaled
//
//   - integer keys are converted to strings
func uniquePathRepresentation(path *gpb.Path) (string, error) {
	b, err := json.Marshal(path.GetElem())
	if err != nil {
		return "", err
	}

	return strings.TrimRight(strings.TrimLeft(string(b), "["), "]"), nil
}
