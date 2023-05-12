package conflict

import (
	"testing"
	"time"
)

func testCreatePhysicalClockConflictResolver() *PhysicalClockConflictResolver {
	return &PhysicalClockConflictResolver{}
}

func (c *PhysicalClockConflictResolver) testCreatePhysicalClock() PhysicalClock {
	return PhysicalClock{timestamp: uint64(time.Now().UnixNano())}
}

func (c *PhysicalClockConflictResolver) testCreatePhysicalClockGivenTimestamp(timestamp uint64) PhysicalClock {
	return PhysicalClock{timestamp: timestamp}
}

func TestPhysicalConcurrentEventsHappenBefore(t *testing.T) {
	r := testCreatePhysicalClockConflictResolver()
	c1 := r.testCreatePhysicalClockGivenTimestamp(10)
	c2 := r.testCreatePhysicalClockGivenTimestamp(20)
	c3 := r.testCreatePhysicalClockGivenTimestamp(30)

	if !c1.HappensBefore(c2) {
		t.Errorf("c1 should happen before c2")
	}
	if !c1.HappensBefore(c3) {
		t.Errorf("c1 should happen before c3")
	}
	if !c2.HappensBefore(c3) {
		t.Errorf("c2 should happen before c3")
	}
}

func TestPhysicalResolverConcurrentKV(t *testing.T) {
	// Create Resolver
	resv := testCreatePhysicalClockConflictResolver()

	// Create new clock
	c1 := resv.NewClock()

	// Create KVs with the same clock
	KV1 := KVFromParts("key", "A", c1)
	KV2 := KVFromParts("key", "B", c1)
	KV3 := KVFromParts("key", "C", c1)

	// Resolver should pick out KV with max key
	maxConflict, _ := resv.ResolveConcurrentEvents(KV1, KV2, KV3)

	if maxConflict.Value != "C" {
		t.Errorf("Value should be C, but is %s\n", maxConflict.Value)
	}
}
