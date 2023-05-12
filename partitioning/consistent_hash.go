package partitioning

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"golang.org/x/exp/slices"
)

// Lookup returns the ID of the replica group to which the specified key is assigned.
// It also returns a hashed version of the key as the second return value.
//
// The replica group ID corresponding to a given key is found by looking up the first
// virtual node that succeeds the hashed key on the ring and returning the replica group ID
// to which this virtual node corresponds. If no replica groups have been added to the ring,
// an error is returned.
func (c *ConsistentHash) Lookup(key string) (id uint64, rewrittenKey string, err error) {
	if len(c.virtualNodes) == 0 {
		return 0, "", errors.New("No replica group")

	}
	hashedKey := c.keyHash(key)
	tmpVirtualNode := new(virtualNode)
	tmpVirtualNode.hash = hashedKey
	targetNode := c.virtualNodes[0]
	for _, virtualNode := range c.virtualNodes {
		// vitualnode < tmpvirtual, next one
		if virtualNodeLess(virtualNode, *tmpVirtualNode) {
			continue
			//if tmpnode == virtual node or vitual node > tmpnode
		} else {
			targetNode = virtualNode
			break
		}

	}

	rewrittenKey = hashToString(hashedKey)
	// if our tmp node bigger all of the vitualnodes hashes, the 0th node is the target node
	return targetNode.id, rewrittenKey, nil

	// TODO(students): [Partitioning] Implement me!
	// return 0, "", errors.New("not implemented")
}

// AddReplicaGroup adds a replica group to the hash ring, returning a list of key ranges that need
// to be reassigned to this new group. Specifically, for each new virtual node, the ring must be
// updated and a corresponding reassignment entry must be created (to be returned).
// If the replica group is already in the ring, this method is a no-op, and a nil slice is
// returned.
//
// The reassignment entry for a given virtual node must specify the key range that needs to be
// moved to the new replica group due to the virtual node (and from where). The length of the
// returned list of reassignments must equal the number of virtual nodes per replica group,
// with one entry corresponding to each virtual node (but in any order).
func (c *ConsistentHash) AddReplicaGroup(id uint64) []Reassignment {
	//check if id to see if it is a existed replica group
	reassignments := make([]Reassignment, 0)
	// fmt.Println("after initialize, the len(reassignments):", len(reassignments))
	for _, virtualNode := range c.virtualNodes {
		if virtualNode.id == id {
			return nil
		}
	}
	// try append first
	// oldVirtualNodes := c.virtualNodes
	newVirtualNodes := c.virtualNodesForGroup(id)

	c.virtualNodes = append(c.virtualNodes, newVirtualNodes...)
	// we sort the updated ring first
	slices.SortFunc(c.virtualNodes, virtualNodeLess)

	for idx, node := range c.virtualNodes {
		if node.id != id {
			continue
		}
		// find the first succeeding old node
		right := slices.IndexFunc(c.virtualNodes, func(n virtualNode) bool {
			return n.id != id && virtualNodeLess(node, n)
		})
		if right == -1 {
			// find the fist old node then, it means the new node is now the first node
			right = slices.IndexFunc(c.virtualNodes, func(n virtualNode) bool {
				return n.id != id
			})
			if right == -1 {
				return nil
			}
		}
		reassignments = append(reassignments, Reassignment{
			From: c.node(right).id,
			To:   id,
			Range: KeyRange{
				Start: hashToString(incrementHash(c.node(idx - 1).hash)),
				End:   hashToString(node.hash),
			},
		})
	}
	return reassignments
	//this is the misleading way of doing it!!
	// oldVirtualNodes := c.virtualNodes
	// newVirtualNodes := c.virtualNodesForGroup(id)
	// //sort the new nodes
	// sort.Slice(newVirtualNodes, func(i, j int) bool {
	// 	return virtualNodeLess(newVirtualNodes[i], newVirtualNodes[j])
	// })
	// if len(c.virtualNodes) == 0 {
	// 	//This is the first replicagroup
	// 	c.virtualNodes = append(c.virtualNodes, newVirtualNodes...)
	// 	return nil
	// }
	// // ok itertae through new virtuals and do update and return reassignment at the same time!
	// //In hindsight, append first then sort then return would be way easier.
	// //but i was in too deep!

	// for _, newNode := range newVirtualNodes {
	// 	// fmt.Println("now the len of nodes :", len(c.virtualNodes))
	// 	if virtualNodeLess(newNode, oldVirtualNodes[0]) {
	// 		continue
	// 		// come back ltr this is the first node now the new node and it is smaller than the first old node
	// 	} else {
	// 		if virtualNodeLess(c.virtualNodes[len(c.virtualNodes)-1], newNode) {
	// 			// new node bigger than all old node
	// 			prevNode := c.virtualNodes[len(c.virtualNodes)-1]
	// 			reassignments = append(reassignments, Reassignment{
	// 				From: oldVirtualNodes[0].id,
	// 				To:   id,
	// 				Range: KeyRange{
	// 					Start: hashToString(incrementHash(prevNode.hash)),
	// 					End:   hashToString(newNode.hash),
	// 				},
	// 			})
	// 			c.virtualNodes = append(c.virtualNodes, newNode)

	// 		} else {
	// 			idx2 := 0
	// 			for idx2 < len(c.virtualNodes) {
	// 				if virtualNodeLess(c.virtualNodes[idx2], newNode) {
	// 					idx2++
	// 					continue
	// 					//if tmpnode == virtual node or vitual node > tmpnode
	// 					//stopped at the first bigger node
	// 				} else {
	// 					prevNode := c.virtualNodes[idx2-1]
	// 					reassignments = append(reassignments, Reassignment{
	// 						From: c.virtualNodes[idx2].id,
	// 						To:   id,
	// 						Range: KeyRange{
	// 							Start: hashToString(incrementHash(prevNode.hash)),
	// 							End:   hashToString(newNode.hash),
	// 						},
	// 					})

	// 					c.virtualNodes = append(c.virtualNodes[:idx2], append([]virtualNode{newNode}, c.virtualNodes[idx2:]...)...)

	// 				}
	// 				break

	// 			}
	// 		}

	// 	}
	// }
	// for _, node := range newVirtualNodes {
	// 	if virtualNodeLess(node, oldVirtualNodes[0]) {
	// 		newRange := KeyRange{
	// 			Start: hashToString(incrementHash(c.virtualNodes[len(c.virtualNodes)-1].hash)),
	// 			End:   hashToString(node.hash),
	// 		}

	// 		newReassignment := Reassignment{
	// 			From:  oldVirtualNodes[0].id,
	// 			To:    id,
	// 			Range: newRange,
	// 		}

	// 		reassignments = append([]Reassignment{newReassignment}, reassignments...)
	// 		// fmt.Println("this is the first new node smaller tthan all oldNode; i append: newNode")
	// 		c.virtualNodes = append([]virtualNode{newVirtualNodes[0]}, c.virtualNodes...)
	// 		sort.Slice(newVirtualNodes, func(i, j int) bool {
	// 			return virtualNodeLess(newVirtualNodes[i], newVirtualNodes[j])
	// 		})
	// 	}

	// }
	// // now check if the first new node is smaller than all the new node, then it is the new starting node
	// // fmt.Println("we added group:", id)
	// // fmt.Println("the reassignment is as follow", reassignments)

	// return reassignments

	// TODO(students): [Partitioning] Implement me!
	// return nil
}

// RemoveReplicaGroup removes a replica group from the hash ring, returning a list of key
// ranges that neeed to be reassigned to other replica groups. If the replica group does
// not exist, this method is a no-op, and an empty slice is returned. It is undefined behavior
// to remove the last replica group from the ring, and this will not be tested.
//
// There must be a reassignment entry for every virtual node of the removed group, specifying
// where its keys should be reassigned. The length of the returned list of reassignments must
// equal the number of virtual nodes per replica group (but in any order). The reassignments
// must also account for every key that was previously assigned to the now removed replica group.
func (c *ConsistentHash) RemoveReplicaGroup(id uint64) []Reassignment {
	if len(c.virtualNodes) == 0 {
		return nil
	}
	//check if id exists
	toBeRemovedNode := make([]virtualNode, 0)
	remainNodes := make([]virtualNode, 0)
	// exist := false
	for _, node := range c.virtualNodes {
		if node.id == id {
			toBeRemovedNode = append(toBeRemovedNode, node)

		} else {
			remainNodes = append(remainNodes, node)
		}

	}
	if len(toBeRemovedNode) == 0 {
		return nil
	}

	reassignments := make([]Reassignment, 0)
	for idx, node := range c.virtualNodes {
		if node.id != id {
			continue
		}
		right := slices.IndexFunc(c.virtualNodes, func(n virtualNode) bool {
			return n.id != id && virtualNodeLess(node, n)
		})
		if right == -1 {
			right = slices.IndexFunc(c.virtualNodes, func(n virtualNode) bool {
				return n.id != id
			})
			if right == -1 {
				// no other group, already considered
				return nil
			}
			//the to be removed node bigger than all other node, and it cannot be the first node in the ring either
			newReassignment := Reassignment{
				From: id,
				To:   c.virtualNodes[right].id,
				Range: KeyRange{
					Start: hashToString(incrementHash(c.virtualNodes[idx-1].hash)),
					End:   hashToString(node.hash),
				},
			}
			reassignments = append(reassignments, newReassignment)

		} else {
			// if the first node got removed
			if idx == 0 {
				newReassignment := Reassignment{
					From: id,
					To:   c.virtualNodes[right].id,
					Range: KeyRange{
						Start: hashToString(incrementHash(c.virtualNodes[len(c.virtualNodes)-1].hash)),
						End:   hashToString(node.hash),
					},
				}
				reassignments = append(reassignments, newReassignment)

			} else {
				newReassignment := Reassignment{
					From: id,
					To:   c.virtualNodes[right].id,
					Range: KeyRange{
						Start: hashToString(incrementHash(c.virtualNodes[idx-1].hash)),
						End:   hashToString(node.hash),
					},
				}
				reassignments = append(reassignments, newReassignment)
			}

		}

	}
	// find the first other-id node succeding the
	c.virtualNodes = remainNodes
	fmt.Println("we remove group:", id)
	fmt.Println("the reassignment is as follow", reassignments)
	// TODO(students): [Partitioning] Implement me!
	return reassignments
}

// ======================================
// DO NOT CHANGE ANY CODE BELOW THIS LINE
// ======================================

// ConsistentHash is a partitioner that implements consistent hashing.
type ConsistentHash struct {
	// virtualNodesPerGroup defines the number of virtual nodes that are created for
	// each replica group.
	virtualNodesPerGroup int

	// virtualNodes defines the hash ring as a sorted list of virtual nodes, starting with the
	// smallest hash value. It must ALWAYS be in ascending sorted order by hash.
	virtualNodes []virtualNode

	// hasher is used to hash all values. Other than pre-defined helpers, this should never be
	// used directly.
	hasher func([]byte) [32]byte
}

// NewConsistentHash creates a new consistent hash partitioner with the default SHA256 hasher.
func NewConsistentHash(virtualNodesPerGroup int) *ConsistentHash {
	return &ConsistentHash{
		virtualNodesPerGroup: virtualNodesPerGroup,
		hasher:               sha256.Sum256,
	}
}

// node returns the virtual node at the specified index.
//
// If the index is out of bounds, it is wrapped using modular arithmetic. For example, an
// index of -1 would map to len(c.virtualNodes)-1.
func (c *ConsistentHash) node(index int) virtualNode {
	clipped := index % len(c.virtualNodes)
	if clipped < 0 {
		clipped += len(c.virtualNodes)
	}
	return c.virtualNodes[clipped]
}

// virtualNodesForGroup returns the virtual nodes for the specified replica group.
// Given the configured parameter, N virtual nodes are created and subsequently returned.
// The virtual nodes are disambiguated by an index that is used when generating their hash.
func (c *ConsistentHash) virtualNodesForGroup(id uint64) []virtualNode {
	var virtualNodes []virtualNode

	for i := 0; i < c.virtualNodesPerGroup; i++ {
		virtualNodeHash := c.virtualNodeHash(id, i)

		virtualNodes = append(virtualNodes, virtualNode{
			id:   id,
			num:  i,
			hash: virtualNodeHash,
		})
	}

	return virtualNodes
}

// virtualNode defines a node in the consistent hash ring. It is a combination
// of the replica group id, the disambiguating virtual number, and the node's hash.
type virtualNode struct {
	id   uint64
	num  int
	hash [32]byte
}

// virtualNodeCmp compares two virtual nodes by their hash, returning -1 if a < b,
// 0 if a == b, and 1 if a > b.
func virtualNodeCmp(a, b virtualNode) int {
	return bytes.Compare(a.hash[:], b.hash[:])
}

// virtualNodeLess compares two virtual nodes by their hash, returning true if and
// only if a < b.
func virtualNodeLess(a, b virtualNode) bool {
	return virtualNodeCmp(a, b) < 0
}

// incrementHash adds 1 to the given hash, wrapping back to 0 if necessary.
func incrementHash(hash [32]byte) [32]byte {
	for i := len(hash) - 1; i >= 0; i-- {
		if hash[i] < math.MaxUint8 {
			hash[i]++
			return hash
		}

		hash[i] = 0
	}
	return hash
}

// hashToString returns the hex string representation of the specified hash. This is useful
// because although we internally represent hashes as byte arrays, we sometimes need to return
// the string hash of a key in our RPC API. It should be used whenever we need to return the
// hash of a key as a string in our API. This includes both specifying reassignemnts and
// creating rewritten keys.
func hashToString(h [32]byte) string {
	return hex.EncodeToString(h[:])
}

// keyHash returns the hash of the specified key.
func (c *ConsistentHash) keyHash(key string) [32]byte {
	hash := c.hasher([]byte(key))

	return hash
}

// virtualNodeHash returns the hash of a virtual node, which is defined by a replica group
// id and number disambiguating different virtual nodes of the same group.
//
// Specifically, the disambiguation number is added to the id before hashing to spread the virtual
// nodes across the ring. Adding, rather than appending, is acceptable since the ids are randomly
// generated and the chance of any conflicts is minimal.
func (c *ConsistentHash) virtualNodeHash(id uint64, virtualNum int) [32]byte {
	virtualID := make([]byte, 8)

	binary.BigEndian.PutUint64(virtualID, id+uint64(virtualNum))

	hash := c.hasher(virtualID)

	return hash
}
