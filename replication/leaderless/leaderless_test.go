package leaderless

import (
	"context"
	"log"
	"math/rand"
	"modist/orchestrator/node"
	pb "modist/proto"
	"modist/replication/conflict"
	"modist/store"
	"strconv"
	"testing"
)

func testCreatePhysicalClockArgs(node *node.Node, w, r int) Args[conflict.PhysicalClock] {
	a := Args[conflict.PhysicalClock]{
		Node:             node,
		W:                w,
		R:                r,
		ConflictResolver: &conflict.PhysicalClockConflictResolver{},
		LocalStore:       &store.Memory[*conflict.KV[conflict.PhysicalClock]]{},
	}
	return a
}

// Test that a cluster with 3 nodes can replicate a key and let us read it back
func TestBasicLeaderless(t *testing.T) {
	// Create nodes and configure them as replicators
	nodes := node.Create([]string{"localhost:1234", "localhost:1235", "localhost:1236"})
	var replicators []*State[conflict.PhysicalClock]

	for _, node := range nodes {
		replicator := Configure[conflict.PhysicalClock](
			testCreatePhysicalClockArgs(node, 2, 2),
		)
		replicators = append(replicators, replicator)
	}

	// Choose the first replicator as coordinator and call ReplicateKey to store key
	key := "foo"
	value := "bar"

	firstReplicator := replicators[0]

	response, err := firstReplicator.ReplicateKey(context.Background(), &pb.PutRequest{
		Key: key, Value: value, Clock: &pb.Clock{Timestamp: 1}})
	if err != nil {
		t.Fatalf("Error while replicating key to node 0: %v", err)
	}

	// Get the key we just stored by calling GetReplicatedKey
	log.Printf("response clock is %v", response.GetClock())
	kv, err := firstReplicator.GetReplicatedKey(context.Background(),
		&pb.GetRequest{Key: key, Metadata: &pb.GetMetadata{Clock: response.GetClock()}})
	if err != nil {
		t.Fatalf("Error while getting key from node 1: %v", err)
	}

	if kv.GetValue() != value {
		t.Fatalf("Value mismatch: expected %v, got %v", value, kv.GetValue())
	}
}

// Read repair brings nodes that have falled behind up-to-date when we do reads. To test that read
// repair works, we can intentionally make a node x fall behind (partition it), do quorum reads
// involving x, and then read only from x. We then make sure that x is as up-to-date as any other
// replica.
func TestBasicReadRepair(t *testing.T) {
	// Create nodes and configure them as replicators
	nodes := node.Create([]string{"localhost:4001", "localhost:4002", "localhost:4003"})
	var replicators []*State[conflict.PhysicalClock]

	for _, node := range nodes {
		replicator := Configure[conflict.PhysicalClock](testCreatePhysicalClockArgs(node, 2, 2))
		replicators = append(replicators, replicator)
	}

	// make replicators[2] fall behind by never including it in any writes
	// NOTE: This can also be done using node.StartPartition(...)
	replicators[0].replicaChooser = func(numreplicas int, exclude []uint64) ([]uint64, error) {
		return []uint64{replicators[1].node.ID}, nil
	}

	replicators[1].replicaChooser = func(numreplicas int, exclude []uint64) ([]uint64, error) {
		return []uint64{replicators[0].node.ID}, nil
	}

	// write lots of data to replicators[0] and replicators[1]
	numItrs := 10
	for i := 0; i < numItrs; i++ {
		pr := &pb.PutRequest{Key: strconv.Itoa(i), Value: strconv.Itoa(numItrs - i), Clock: &pb.Clock{Timestamp: 1}}
		replicators[i%2].ReplicateKey(context.Background(), pr)
	}

	// force the reads from replicators[0] or [1] to include replicators[2]
	replicators[1].replicaChooser = func(numReplicas int, exclude []uint64) ([]uint64, error) {
		return []uint64{replicators[2].node.ID}, nil
	}

	// get replicated key with the second node as the coordinator
	// whose read quorum we hardcoded to include the partitioned node
	for i := 0; i < numItrs; i++ {
		log.Printf("requesting key")
		replicators[1].GetReplicatedKey(context.Background(),
			&pb.GetRequest{Key: strconv.Itoa(i), Metadata: &pb.GetMetadata{Clock: &pb.Clock{Timestamp: 1}}})
	}

	// finally, ensure that replicators[2] has every key itself (by setting r = 1)
	for i := 0; i < numItrs; i++ {
		got, ok := replicators[2].localStore.Get(strconv.Itoa(i))
		expected := strconv.Itoa(numItrs - i)

		if !ok {
			t.Errorf("key %d not found in local store", i)
		} else if got.Value != expected {
			t.Errorf("value mismatch: expected %s for key %d, got %s", expected, i, got)
		}
	}
}

// Do a bunch of read and writes in a cluster of 10 nodes
// Then we check if nodes with the corresponding keys have the correct value
func TestEventualConsistency(t *testing.T) {
	// Create nodes and configure them as replicators
	numNodes := 10
	var addressStrings []string
	for i := 0; i < numNodes; i++ {
		port := 5100 + i
		addr := "localhost:" + strconv.Itoa(port)
		addressStrings = append(addressStrings, addr)
	}
	nodes := node.Create(addressStrings)
	var replicators []*State[conflict.PhysicalClock]

	for _, node := range nodes {
		replicator := Configure[conflict.PhysicalClock](
			testCreatePhysicalClockArgs(node, 2, 2),
		)
		replicators = append(replicators, replicator)
	}

	// write lots of data, with random nodes selected as coordinator
	numItrs := 10
	var responseClocks []*pb.Clock // we will need to update our client clock for read
	for i := 0; i < numItrs; i++ {
		pr := &pb.PutRequest{Key: strconv.Itoa(i), Value: strconv.Itoa(numItrs - i), Clock: &pb.Clock{Timestamp: 1}}
		response, _ := replicators[rand.Intn(numNodes)].ReplicateKey(context.Background(), pr)
		responseClocks = append(responseClocks, response.GetClock())
	}

	// read data from random node, with clock we got from write's response
	for i := 0; i < numItrs; i++ {
		log.Printf("requesting key")
		replicatorIdx := rand.Intn(numNodes)
		replicators[replicatorIdx].GetReplicatedKey(context.Background(),
			&pb.GetRequest{Key: strconv.Itoa(i), Metadata: &pb.GetMetadata{Clock: responseClocks[replicatorIdx]}})
	}

	// check if nodes with the corresponding key have the correct value
	for nodeNum := 0; nodeNum < numNodes; nodeNum++ {
		for i := 0; i < numItrs; i++ {
			got, ok := replicators[2].localStore.Get(strconv.Itoa(i))
			expected := strconv.Itoa(numItrs - i)

			// we do not need to care about nodes where keys are not written to
			if ok && got.Value != expected {
				t.Errorf("Replicator %d: value mismatch: expected %s for key %d, got %s", nodeNum, expected, i, got)
			}
		}
	}
}

// Basically the same test as above, but we now add in partition!
// we will use 3 nodes in this cluster
func TestEventualConsistencyPartitioned(t *testing.T) {
	// Create nodes and configure them as replicators
	numNodes := 3
	var addressStrings []string
	for i := 0; i < numNodes; i++ {
		port := 5200 + i
		addr := "localhost:" + strconv.Itoa(port)
		addressStrings = append(addressStrings, addr)
	}
	nodes := node.Create(addressStrings)
	var replicators []*State[conflict.PhysicalClock]

	for _, node := range nodes {
		replicator := Configure[conflict.PhysicalClock](
			testCreatePhysicalClockArgs(node, 2, 2),
		)
		replicators = append(replicators, replicator)
	}

	// write lots of data, with random nodes selected as coordinator
	numItrs := 10
	var responseClocks []*pb.Clock // we will need to update our client clock for read
	for i := 0; i < numItrs-5; i++ {
		pr := &pb.PutRequest{Key: strconv.Itoa(i), Value: strconv.Itoa(numItrs - i), Clock: &pb.Clock{Timestamp: 1}}
		response, _ := replicators[rand.Intn(numNodes)].ReplicateKey(context.Background(), pr)
		responseClocks = append(responseClocks, response.GetClock())
	}
	// Isolate node 0
	node.StartPartition(nodes, nodes[1], nodes[2])
	node.StartPartition(nodes, nodes[0], nodes[0])

	// keep writing
	for i := numItrs - 5; i < numItrs; i++ {
		pr := &pb.PutRequest{Key: strconv.Itoa(i), Value: strconv.Itoa(numItrs - i), Clock: &pb.Clock{Timestamp: 1}}
		response, _ := replicators[rand.Intn(numNodes)].ReplicateKey(context.Background(), pr)
		responseClocks = append(responseClocks, response.GetClock())
	}

	// All nodes can communicate with each other again
	node.EndPartition(nodes, nodes...)

	// read data from random node, with clock we got from write's response
	for i := 0; i < numItrs; i++ {
		log.Printf("requesting key")
		replicatorIdx := rand.Intn(numNodes)
		replicators[replicatorIdx].GetReplicatedKey(context.Background(),
			&pb.GetRequest{Key: strconv.Itoa(i), Metadata: &pb.GetMetadata{Clock: responseClocks[replicatorIdx]}})
	}

	// check if nodes with the corresponding key have the correct value
	for nodeNum := 0; nodeNum < numNodes; nodeNum++ {
		for i := 0; i < numItrs; i++ {
			got, ok := replicators[2].localStore.Get(strconv.Itoa(i))
			expected := strconv.Itoa(numItrs - i)

			// we do not need to care about nodes where keys are not written to
			if ok && got.Value != expected {
				t.Errorf("Replicator %d: value mismatch: expected %s for key %d, got %s", nodeNum, expected, i, got)
			}
		}
	}
}
