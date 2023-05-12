package tapestry

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSampleTapestrySetup(t *testing.T) {
	tap, _ := MakeTapestries(true, "1", "3", "5", "7") //Make a tapestry with these ids
	fmt.Printf("length of tap %d\n", len(tap))
	KillTapestries(tap[1], tap[2]) //Kill off two of them.
	resp, _ := tap[0].FindRoot(
		context.Background(),
		CreateIDMsg("2", 0),
	) //After killing 3 and 5, this should route to 7
	if resp.Next != tap[3].Id.String() {
		t.Errorf("Failed to kill successfully")
	}

}

func TestSampleTapestrySearch(t *testing.T) {
	tap, _ := MakeTapestries(true, "100", "456", "1234") //make a sample tap
	tap[1].Store("look at this lad", []byte("an absolute unit"))
	result, err := tap[0].Get("look at this lad") //Store a KV pair and try to fetch it
	fmt.Println(err)
	if !bytes.Equal(result, []byte("an absolute unit")) { //Ensure we correctly get our KV
		t.Errorf("Get failed")
	}
}

func TestSampleTapestryAddNodes(t *testing.T) {
	// Need to use this so that gRPC connections are set up correctly
	tap, delayNodes, _ := MakeTapestriesDelayConnecting(
		true,
		[]string{"1", "5", "9"},
		[]string{"8", "12"},
	)

	// Add some tap nodes after the initial construction
	for _, delayNode := range delayNodes {
		args := Args{
			Node:      delayNode,
			Join:      true,
			ConnectTo: tap[0].RetrieveID(tap[0].Id),
		}
		tn := Configure(args).tapestryNode
		tap = append(tap, tn)
		time.Sleep(1000 * time.Millisecond) //Wait for availability
	}

	resp, _ := tap[1].FindRoot(context.Background(), CreateIDMsg("7", 0))
	if resp.Next != tap[3].Id.String() {
		t.Errorf("Addition of node failed")
	}
}

func TestAddRouteRemoveOldNode(t *testing.T) {
	tap, _ := MakeTapestries(true, "3f93", "1c42", "2fe4", "437e", "5c21", "65bb", "705b", "8887", "93cb", "c3ca", "d340", "e9ce", "309c", "362d", "3c6f") //Make a tapestry with these ids
	// will trigger RemoveBackPointers since there will be more than SLOTSIZE nodes in a slot

	fmt.Printf("length of tap %d\n", len(tap))

	resp, _ := tap[3].FindRoot(context.Background(), CreateIDMsg("3f92", 0))
	if resp.Next != tap[0].Id.String() {
		t.Errorf("failed TestAddRouteRemoveOldNode")
	}	
}

func TestRemoveRootDuringFindRoot(t *testing.T) {
	tap, _ := MakeTapestries(true, "3f93", "1c42", "2fe4", "437e", "5c21", "65bb", "705b", "8887", "93cb", "c3ca", "d340", "e9ce", "309c", "362d", "3c6f") //Make a tapestry with these ids

	fmt.Printf("length of tap %d\n", len(tap))
	
	go func() {
		time.Sleep(1000 * time.Microsecond)
		tap[3].Store("3f92", []byte("values la la la")) // trigger find root retries since 3f93 is killed
		time.Sleep(5 * time.Millisecond)
		resp, _ := tap[3].FindRoot(context.Background(), CreateIDMsg("3f92", 0))
		if resp.Next == tap[0].Id.String() {
			t.Errorf("should be routed to a different root!")
		}
	}()
	time.Sleep(1500 * time.Microsecond)
	tap[0].Kill()
}

func TestFindRootError(t *testing.T) {
	tap, _, _ := MakeTapestriesDelayConnecting(
		true,
		[]string{"1c42", "2fe4", "437e", "5c21", "65bb", "705b", "8887", "93cb", "c3ca", "d340", "e9ce", "309c", "362d", "3c6f"},
		[]string{"3f93"}, // we don't connect 3f93
	)
	tap[3].Store("3f92", []byte("values la la la")) // trigger find root retries since 3f93 is not connected
	resp, _ := tap[3].FindRoot(context.Background(), CreateIDMsg("3f92", 0))
	if resp.Next != tap[11].Id.String() {
		t.Errorf("failed TestFindRootError %s", resp.Next)
	}	
}

func TestRepublish(t *testing.T) {
	tap, _ := MakeTapestries(true, "3f93", "1c42", "2fe4", "437e", "5c21", "65bb", "705b", "8887", "93cb", "c3ca", "d340", "e9ce", "309c", "362d", "3c6f") //Make a tapestry with these ids

	tap[3].Store("3f92", []byte("values la la la")) 
	time.Sleep(15 * time.Second)
	resp, _ := tap[3].FindRoot(context.Background(), CreateIDMsg("3f92", 0))
	if resp.Next != tap[0].Id.String() {
		t.Errorf("failed TestAddRouteRemoveOldNode")
	}
}

func TestExit(t *testing.T) {
	tap, _ := MakeTapestries(true, "3f93", "1c42", "2fe4", "437e", "5c21", "65bb", "705b", "8887", "93cb", "c3ca", "d340", "e9ce", "309c", "362d", "3c6f") //Make a tapestry with these ids
	tap[0].Leave()
	time.Sleep(time.Second)
	resp, _ := tap[3].FindRoot(context.Background(), CreateIDMsg("3f92", 0))
	if resp.Next == tap[0].Id.String() {
		t.Errorf("failed TestExit")
	}
}