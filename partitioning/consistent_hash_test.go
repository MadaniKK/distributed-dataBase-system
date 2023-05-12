package partitioning

import (
	"encoding/binary"
	"fmt"
	"testing"

	"golang.org/x/exp/slices"
)

// checkLookup performs a lookup of key using the provided consistent hash partitioner,
// ensuring that there is no error, the returned id matches what is expected, and the
// rewritten key is a hash of the looked-up key.
func checkLookup(t *testing.T, msg string, c *ConsistentHash, key string, id uint64) {
	t.Helper()

	gotID, gotRewrittenKey, err := c.Lookup(key)
	rewrittenKey := hashToString(c.keyHash(key))
	if err != nil {
		t.Errorf("%s: Returned an error: %v", msg, err)
	} else if gotID != id {
		t.Errorf("%s: Returned the wrong shard: expected %d, got %d\nThe hashed key is %s\n Here are the virtual nodes in the assigner: %+v\n\n", msg, id, gotID, rewrittenKey, c.virtualNodes)
	} else if gotRewrittenKey != rewrittenKey {
		t.Errorf("%s: Returned the wrong rewritten key: expected %s, got %s", msg, rewrittenKey, gotRewrittenKey)
	}
}

// identityHasher returns the last 32 bytes of the input as a 32-byte array, padding with
// zeroes if necessary.
func identityHasher(b []byte) [32]byte {
	var out [32]byte

	bIndex := len(b) - 1
	for i := len(out) - 1; i >= 0; i-- {
		if bIndex < 0 {
			continue
		}
		out[i] = b[bIndex]
		bIndex--
	}
	return out
}

func newVirtualNode(c *ConsistentHash, id uint64, virtualNum int) virtualNode {
	return virtualNode{
		id:   id,
		num:  virtualNum,
		hash: c.virtualNodeHash(id, virtualNum),
	}
}

func TestConsistentHash_Lookup_SimpleIdentity(t *testing.T) {
	c := NewConsistentHash(2)
	c.hasher = identityHasher

	c.virtualNodes = []virtualNode{
		newVirtualNode(c, 1, 0),
		newVirtualNode(c, 1, 1),
		newVirtualNode(c, 50, 0),
		newVirtualNode(c, 50, 1),
	}
	slices.SortFunc(c.virtualNodes, virtualNodeLess)

	byteKey := make([]byte, 8)

	binary.BigEndian.PutUint64(byteKey, 2)
	checkLookup(t, "Lookup(10)", c, string(byteKey), 1)

	binary.BigEndian.PutUint64(byteKey, 10)
	checkLookup(t, "Lookup(10)", c, string(byteKey), 50)

	binary.BigEndian.PutUint64(byteKey, 50)
	checkLookup(t, "Lookup(50)", c, string(byteKey), 50)

	binary.BigEndian.PutUint64(byteKey, 51)
	checkLookup(t, "Lookup(51)", c, string(byteKey), 50)
}

func TestBasicAddtoGroup(t *testing.T) {
	c := NewConsistentHash(2)
	c.hasher = identityHasher

	c.virtualNodes = []virtualNode{
		newVirtualNode(c, 1, 0),
		newVirtualNode(c, 1, 1),
		newVirtualNode(c, 50, 0),
		newVirtualNode(c, 50, 1),
	}
	slices.SortFunc(c.virtualNodes, virtualNodeLess)
	reassignments := c.AddReplicaGroup(20)
	if len(reassignments) == 0 {
		fmt.Println("sth worng")
	}
	reassignments = c.AddReplicaGroup(50)
	if reassignments != nil {
		t.Errorf("Adding existed replica group should return nil")
	}
	fmt.Println(len(reassignments))
	fmt.Println(reassignments)
	// for _, node := range c.virtualNodes {
	// 	fmt.Println(node.id, node.num, hashToString(node.hash))

	// }

}

func TestBasicRemoveReplicaGroup(t *testing.T) {
	c := NewConsistentHash(2)
	c.hasher = identityHasher
	result := c.RemoveReplicaGroup(1)
	if result != nil {
		t.Errorf("should return nil when removing from an empty ring")
	}

	c.virtualNodes = []virtualNode{
		newVirtualNode(c, 1, 0),
		newVirtualNode(c, 1, 1),
		newVirtualNode(c, 50, 0),
		newVirtualNode(c, 50, 1),
		newVirtualNode(c, 30, 0),
		newVirtualNode(c, 30, 1),
		newVirtualNode(c, 4, 0),
		newVirtualNode(c, 4, 1),
	}
	slices.SortFunc(c.virtualNodes, virtualNodeLess)
	reassignments := c.RemoveReplicaGroup(1)
	reassignments = c.RemoveReplicaGroup(30)
	reassignments = c.RemoveReplicaGroup(4)

	if len(reassignments) == 0 {
		t.Errorf("sth wrong with reareassignments")
	}
	reassignments = c.RemoveReplicaGroup(2)
	if reassignments != nil {
		t.Errorf("Removing non-existed replica group should return nil")

	}
	// fmt.Println(len(reassignments))
	// fmt.Println(reassignments)

}
func TestLookUpEdgeCase(t *testing.T) {
	c := NewConsistentHash(2)
	c.hasher = identityHasher
	_, _, err := c.Lookup("key")
	if err == nil {
		t.Errorf("should return error when looking up in an empty ring!")
	}
}
func TestBasicIntegration(t *testing.T) {
	c := NewConsistentHash(2)
	c.hasher = identityHasher

	var virtualNodes []virtualNode
	for i := 0; i < 1000; i++ {
		virtualNodes = append(virtualNodes, newVirtualNode(c, uint64(i), 0))
		virtualNodes = append(virtualNodes, newVirtualNode(c, uint64(i), 1))
	}

	c.virtualNodes = virtualNodes

	slices.SortFunc(c.virtualNodes, virtualNodeLess)

	var reassignments []Reassignment
	for i := 0; i < 1000; i++ {
		reassignments = c.RemoveReplicaGroup(uint64(i))
	}

	if len(reassignments) != 0 {
		t.Errorf("sth wrong with reareassignments")
	}

	fmt.Println(len(reassignments))
	fmt.Println(reassignments)
}
