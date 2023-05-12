package conflict

import (
	"errors"
	"fmt"
	"log"
	"math"
	"modist/orchestrator/node"
	pb "modist/proto"
	"sync"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	// "google.golang.org/protobuf/internal/errors"
)

// VersionVectorClock is the Clock that we use to implement causal consistency.
type VersionVectorClock struct {
	// Map from node ID to the associated counter. If a node ID isn't in the map, then its counter
	// is considered to be 0 (we don't automatically populate node IDs to save memory).
	vector map[uint64]uint64
}

// NewVersionVectorClock returns an initialized VersionVectorClock.
func NewVersionVectorClock() VersionVectorClock {
	return VersionVectorClock{vector: make(map[uint64]uint64)}
}

// Proto converts a VersionVectorClock into a clock that can be sent in an RPC.
func (v VersionVectorClock) Proto() *pb.Clock {
	p := &pb.Clock{
		Vector: v.vector,
	}
	return p
}

func (v VersionVectorClock) String() string {
	return fmt.Sprintf("%v", v.vector)
}

func (v VersionVectorClock) Equals(other Clock) bool {
	otherVector := other.(VersionVectorClock)
	return maps.Equal(v.vector, otherVector.vector)
}

// HappensBefore returns whether v happens before other. With version vectors, this happens when
// two conditions are met:
//   - For every nodeID in v, other has a counter greater than or equal to v's counter for that
//     node
//   - The vectors are not equal.
//
// Remember that nodeIDs that are not in a vector have an implicit counter of 0.
func (v VersionVectorClock) HappensBefore(other Clock) bool {
	otherVector := other.(VersionVectorClock)
	if v.Equals(otherVector) {
		return false
	}

	// if all otherclock's entries are all zero
	isZero := true
	for _, value := range otherVector.vector {
		if value != 0 {
			isZero = false
		}
	}

	if isZero {
		return false
	}

	for key1, value1 := range v.vector {
		value2, ok := otherVector.vector[key1]
		if !ok {
			// local clock has a key that other clock does not have, return true
			if value1 != 0 {
				return false
			}
		} else {
			// if local key also exist in new clock
			if value1 > value2 {
				return false
			}

		}

	}

	// TODO(students): [Clocks & Conflict Resolution] Implement me!
	return true
}

// Version vector implementation of a ConflictResolver. Might need to keep some state in here
// so that we can always give an up-to-date version vector.
type VersionVectorConflictResolver struct {
	// The node ID on which this conflict resolver is running. Used so that when a message is
	// received, vector[nodeID] can be incremented.
	nodeID uint64

	// mu guards vector
	mu sync.Mutex
	// This node's current clock
	vector map[uint64]uint64
}

// NewVersionVectorConflictResolver() returns an initialized VersionVectorConflictResolver{}
func NewVersionVectorConflictResolver() *VersionVectorConflictResolver {
	return &VersionVectorConflictResolver{vector: make(map[uint64]uint64)}
}

// ReplicatorDidStart initializes the VersionVectorConflictResolver using node metadata
func (v *VersionVectorConflictResolver) ReplicatorDidStart(node *node.Node) {
	v.nodeID = node.ID
	v.vector[v.nodeID] = 0

	log.Printf("version vector conflict resolver initializing itself")
}

// Finds the max of two ordered entities, x and y. constraints.Ordered is an alias for Integers
// and Floats.
func max[T constraints.Ordered](x T, y T) T {
	if x > y {
		return x
	}
	return y
}

// OnMessageReceive is called whenever the underlying node receives an RPC with a clock. As per
// the version-vector algorithm, this function does the following:
//   - Sets the current node's clock to be the element-wise max of itself and the given clock
//   - Increments its own nodeID in vector
//
// Remember thread-safety when modifying fields of v, since multiple messages could be received at
// the same time!
func (v *VersionVectorConflictResolver) OnMessageReceive(clock VersionVectorClock) {

	v.mu.Lock()
	defer v.mu.Unlock()

	newVector := make(map[uint64]uint64)
	// the biggest clock might have keys that not exist in the local clock
	for key, value := range clock.vector {
		newVector[key] = value

	}

	for key, value := range v.vector {
		newVector[key] = uint64(math.Max(float64(value), float64(newVector[key])))
	}
	v.vector = newVector

	//increase its own nodeID  in vector
	v.vector[v.nodeID]++

	return

	// TODO(students): [Clocks & Conflict Resolution] Implement me!
}

// OnMessageSend is called before an RPC is sent to any other node. The version vector should be
// incremented for the local node.
func (v *VersionVectorConflictResolver) OnMessageSend() {

	v.mu.Lock()
	defer v.mu.Unlock()

	v.vector[v.nodeID]++

	// TODO(students): [Clocks & Conflict Resolution] Implement me!
}

func (v *VersionVectorConflictResolver) OnEvent() {
	panic("disregard; not yet implemented in modist")
}

// NewClock creates a new VersionVectorClock by using v's vector.
//
// Note that maps in Golang are implicit pointers, so you should deep-copy the map before
// returning it.
func (v *VersionVectorConflictResolver) NewClock() VersionVectorClock {
	v.mu.Lock()
	defer v.mu.Unlock()

	newClock := make(map[uint64]uint64, len(v.vector))

	for key, value := range v.vector {
		newClock[key] = value
	}

	// TODO(students): [Clocks & Conflict Resolution] Implement me!
	return VersionVectorClock{vector: newClock}
}

// ZeroClock returns a clock that happens before (or is concurrent with) all other clocks.
func (v *VersionVectorConflictResolver) ZeroClock() VersionVectorClock {
	return VersionVectorClock{vector: map[uint64]uint64{}}
}

// ResolveConcurrentEvents is run when we have several key-value pairs with the same keys, all
// with concurrent clocks (i.e. no version vector happens before any other version vector). To
// converge to one value, we must choose a "winner" among these key-value pairs. Like in
// physical.go, we choose the key-value with the highest lexicographic value.
//
// Additionally, the returned key-value must have a clock that is higher than all the given
// key-value pairs. You can construct a new clock for the returned key by merging the clocks of
// the conflicts slice together (merging two version vectors means computing the element-wise
// max).
//
// You should return an error if no conflicts are given.
func (v *VersionVectorConflictResolver) ResolveConcurrentEvents(
	conflicts ...*KV[VersionVectorClock]) (*KV[VersionVectorClock], error) {
	if len(conflicts) == 0 {
		return nil, errors.New("There is no conflicts")
	}

	maxConflict := new(KV[VersionVectorClock])

	newClock := NewVersionVectorClock()
	//  produce a clock that is that has the bigggest clocks of all clocks!
	for _, conflict := range conflicts {
		for key, value := range conflict.Clock.vector {

			newClock.vector[key] = uint64(math.Max(float64(value), float64(newClock.vector[key])))

		}
	}
	maxConflict.Clock = newClock

	// var maxValue string

	for _, conflict := range conflicts {
		if conflict.Value > maxConflict.Value {
			maxConflict.Value = conflict.Value
			maxConflict.Key = conflict.Key
		}
	}

	// // find the biggest values

	// //  first find the max values in all the conflicts clock vectors! element wise!!
	// for _, conflict := range conflicts[1:] {
	// 	if conflict.Value > maxConflict.Value {

	// 		newClock := NewVersionVectorClock()
	// 		// the biggest conflicts clocks might have keys that not exist in the current maxconflict clock
	// 		for key, value := range conflict.Clock.vector {
	// 			newClock.vector[key] = value

	// 		}

	// 		for key, value := range maxConflict.Clock.vector {
	// 			newClock.vector[key] = uint64(math.Max(float64(value), float64(newClock.vector[key])))
	// 		}

	// 		maxConflict.Key = conflict.Key
	// 		maxConflict.Clock = newClock
	// 		maxConflict.Value = conflict.Value
	// 	}

	// }

	// TODO(students): [Clocks & Conflict Resolution] Implement me!
	return maxConflict, nil
}
