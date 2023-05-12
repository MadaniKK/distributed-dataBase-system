package conflict

import (
	"reflect"
	"testing"
)

func TestVectorConcurrentEventsDoNotHappenBefore(t *testing.T) {
	v1 := NewVersionVectorClock()
	v2 := NewVersionVectorClock()

	v1.vector[0] = 0
	v1.vector[3] = 2
	v2.vector[0] = 1

	if v2.HappensBefore(v1) {
		t.Errorf("v2 does not happen before v1 due to v1[3]")
	}
	if v1.HappensBefore(v2) {
		t.Errorf("v1 does not happen before v2 due to v2[0] > v1[0]")
	}
}

// Edge case where keys X exist in v1 but not v2, but X has value 0 in v1
func TestVectorHappensBeforeEdge(t *testing.T) {
	v1 := NewVersionVectorClock()
	v2 := NewVersionVectorClock()

	v1.vector = map[uint64]uint64{
		1: 0,
		2: 0,
		3: 0,
	}

	v2.vector = map[uint64]uint64{
		2: 2,
	}

	if v2.HappensBefore(v1) {
		t.Errorf("v1 should happen before v2: v1 = {1:0, 2:0, 3:0} while v2 = {2:2}")
	}
}

func TestVectorResolverConcurrentKV(t *testing.T) {
	// Create Resolver and initialize
	v := NewVersionVectorConflictResolver()
	v.nodeID = 3
	v.vector[v.nodeID] = 0

	// Create KVs (we don't care about clocks in this test)
	KV1 := KVFromParts("key", "A", NewVersionVectorClock())
	KV2 := KVFromParts("key", "B", NewVersionVectorClock())
	KV3 := KVFromParts("key", "C", NewVersionVectorClock())

	// Resolver should pick out KV with max key
	maxConflict, _ := v.ResolveConcurrentEvents(KV1, KV2, KV3)

	if maxConflict.Value != "C" {
		t.Errorf("Value should be C, but is %s\n", maxConflict.Value)
	}
}

func TestVectorResolverConcurrentNewClock(t *testing.T) {
	// Create Resolver and initialize
	v := NewVersionVectorConflictResolver()
	v.nodeID = 3
	v.vector[v.nodeID] = 0

	// Create clocks for KVs
	clock1 := NewVersionVectorClock()
	clock1.vector = map[uint64]uint64{
		1: 0,
		3: 1,
		4: 5,
	}
	clock2 := NewVersionVectorClock()
	clock2.vector = map[uint64]uint64{
		2: 4,
		3: 2,
		5: 0,
	}
	clock3 := NewVersionVectorClock()
	clock3.vector = map[uint64]uint64{
		1: 1,
		2: 4,
		3: 0,
	}
	// Create KVs
	KV1 := KVFromParts("key", "A", clock1)
	KV2 := KVFromParts("key", "B", clock2)
	KV3 := KVFromParts("key", "C", clock3)

	// Apply resolver and check new clock
	maxConflict, _ := v.ResolveConcurrentEvents(KV1, KV2, KV3)

	expRes := map[uint64]uint64{
		1: 1,
		2: 4,
		3: 2,
		4: 5,
		5: 0,
	}
	newClockVector := maxConflict.Clock.vector
	if !reflect.DeepEqual(newClockVector, expRes) {
		t.Errorf("New clock is wrong!")
	}
}

func TestReceiveMessage(t *testing.T) {
	// Create Resolver and initialize local version vector
	v := NewVersionVectorConflictResolver()
	v.nodeID = 3
	v.vector = map[uint64]uint64{
		1: 1,
		2: 4,
		3: 0,
	}

	// Create new clock
	clock1 := NewVersionVectorClock()
	clock1.vector = map[uint64]uint64{
		1: 0,
		4: 5,
	}
	// Call receive message
	v.OnMessageReceive(clock1)

	// Check new local vector
	expRes := map[uint64]uint64{
		1: 1,
		2: 4,
		3: 1,
		4: 5,
	}
	if !reflect.DeepEqual(v.vector, expRes) {
		t.Errorf("Local clock is incorrect after receiving message")
	}
}
