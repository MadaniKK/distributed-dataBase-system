/*
 *  Brown University, CS138, Spring 2023
 *
 *  Purpose: Defines global constants and functions to create and join a new
 *  node into a Tapestry mesh, and functions for altering the routing table
 *  and backpointers of the local node that are invoked over RPC.
 */

package tapestry

import (
	"context"
	"fmt"
	"log"
	"modist/orchestrator/node"
	pb "modist/proto"
	"sort"
	"time"
)

// BASE is the base of a digit of an ID.  By default, a digit is base-16.
const BASE = 16

// DIGITS is the number of digits in an ID.  By default, an ID has 40 digits.
const DIGITS = 40

// RETRIES is the number of retries on failure. By default we have 3 retries.
const RETRIES = 3

// K is neigborset size during neighbor traversal before fetching backpointers. By default this has a value of 10.
const K = 10

// SLOTSIZE is the size each slot in the routing table should store this many nodes. By default this is 3.
const SLOTSIZE = 3

// REPUBLISH is object republish interval for nodes advertising objects.
const REPUBLISH = 10 * time.Second

// TIMEOUT is object timeout interval for nodes storing objects.
const TIMEOUT = 25 * time.Second

// TapestryNode is the main struct for the local Tapestry node. Methods can be invoked locally on this struct.
type TapestryNode struct {
	Node *node.Node // Node that this Tapestry node is part of
	Id   ID         // ID of node in the form of a slice (makes it easier to iterate)

	Table          *RoutingTable // The routing table
	Backpointers   *Backpointers // Backpointers to keep track of other nodes that point to us
	LocationsByKey *LocationMap  // Stores keys for which this node is the root
	blobstore      *BlobStore    // Stores blobs on the local node

	// Observability
	log *log.Logger

	// These functions are the internal, private RPCs for a routing node using Tapestry
	pb.UnsafeTapestryRPCServer
}

func (local *TapestryNode) String() string {
	return fmt.Sprint(local.Id)
}

// Called in tapestry initialization to create a tapestry node struct
func newTapestryNode(node *node.Node) *TapestryNode {
	tn := new(TapestryNode)

	tn.Node = node
	tn.Id = MakeID(node.ID)
	tn.Table = NewRoutingTable(tn.Id)
	tn.Backpointers = NewBackpointers(tn.Id)
	tn.LocationsByKey = NewLocationMap()
	tn.blobstore = NewBlobStore()

	tn.log = node.Log

	return tn
}

// Start Tapestry Node
func StartTapestryNode(node *node.Node, connectTo uint64, join bool) (tn *TapestryNode, err error) {
	// Create the local node
	tn = newTapestryNode(node)

	tn.log.Printf("Created tapestry node %v\n", tn)

	grpcServer := tn.Node.GrpcServer
	pb.RegisterTapestryRPCServer(grpcServer, tn)

	// If specified, connect to the provided ID
	if join {
		// If provided ID doesn't exist, return an error
		if _, ok := node.PeerConns[connectTo]; !ok {
			return nil, fmt.Errorf(
				"Error joining Tapestry node with id %v; Unable to find node %v in peerConns",
				connectTo,
				connectTo,
			)
		}

		err = tn.Join(MakeID(connectTo))
		if err != nil {
			tn.log.Printf(err.Error())
			return nil, err
		}
	}

	return tn, nil
}

// removeDuplicatesAndTrimToK is a helper function work as its name. Used in join for backpoint traversal
func (local *TapestryNode) removeDuplicatesAndTrimToK(neighbors []ID) []ID {
	if len(neighbors) <= K {
		return neighbors
	} else {
		sort.Slice(neighbors, func(i, j int) bool {
			// IdA, _ := ParseID(neighbors[i])
			// IdB, _ := ParseID(neighbors[i])
			return local.Id.Closer(neighbors[i], neighbors[j])
		})
		return neighbors[0:K]
	}
}
func UniqueIDResults(IDSlices ...[]ID) []ID {
	seen := make(map[ID]bool)
	result := []ID{}

	for _, slice := range IDSlices {
		for _, str := range slice {
			if _, ok := seen[str]; !ok {
				seen[str] = true
				result = append(result, str)
			}
		}
	}

	return result
}

// Join is invoked when starting the local node, if we are connecting to an existing Tapestry.
//
// - Find the root for our node's ID
// - Call AddNode on our root to initiate the multicast and receive our initial neighbor set. Add them to our table.
// - Iteratively get backpointers from the neighbor set for all levels in range [0, SharedPrefixLength]
// and populate routing table
func (local *TapestryNode) Join(remoteNodeId ID) error {
	local.log.Println("Joining tapestry node", remoteNodeId)
	// 1. Find the root node
	// 2. add new id to the routing table of root node and other nodes
	// 3. add the initial Neighbours to the routing table of new node
	//Iteratively get backpointers from the neighbor set and populate routing table and neighbors set

	// Route to our root
	rootIdPtr, err := local.FindRootOnRemoteNode(remoteNodeId, local.Id)
	if err != nil {
		return fmt.Errorf("Error joining existing tapestry node %v, reason: %v", remoteNodeId, err)
	}
	rootId := *rootIdPtr

	// Add ourselves to our root by invoking AddNode on the remote node
	nodeMsg := &pb.NodeMsg{
		Id: local.Id.String(),
	}

	conn := local.Node.PeerConns[local.RetrieveID(rootId)]
	rootNode := pb.NewTapestryRPCClient(conn)
	resp, err := rootNode.AddNode(context.Background(), nodeMsg)
	if err != nil {
		return fmt.Errorf("Error adding ourselves to root node %v, reason: %v", rootId, err)
	}

	local.log.Println("Joining!!!!!")

	// Add the neighbors to our local routing table.
	neighborIds, err := stringSliceToIds(resp.Neighbors)
	if err != nil {
		return fmt.Errorf("Error parsing neighbor IDs, reason: %v", err)
	}
	if len(neighborIds) == 0 {
		local.log.Println("neighborIds len is 0")
		// return nil
	}

	for _, neighborId := range neighborIds {
		local.AddRoute(neighborId)
	}

	local.log.Println("Joining!!!!!This is where my code starts")

	currentNeighbors := neighborIds

	level := SharedPrefixLength(rootId, local.Id) - 1
	for level >= 0 {
		currentNeighbors = local.removeDuplicatesAndTrimToK(currentNeighbors)
		// check here for the scoping
		next := currentNeighbors

		resultChan := make(chan []ID)
		for _, id := range currentNeighbors {
			go func() {
				conn := local.Node.PeerConns[local.RetrieveID(id)]
				neighborNode := pb.NewTapestryRPCClient(conn)
				reqMsg := &pb.BackpointerRequest{
					From:  local.String(),
					Level: int32(level),
				}
				temp, err := neighborNode.GetBackpointers(context.Background(), reqMsg)
				if err != nil {
					var neighbors []string
					neighbors = append(neighbors, id.String())
					pbNeighbor := &pb.Neighbors{
						Neighbors: neighbors,
					}
					// badNeighbors := []*pb.Neighbors{}
					// badNeighbors = append(badNeighbors, pbNeighbor)
					// use a slice  remove later
					local.RemoveBadNodes(context.Background(), pbNeighbor)
				} else {
					neigbors, _ := stringSliceToIds(temp.Neighbors)
					resultChan <- neigbors
					// next = append(next, temp.Neighbors...)
				}
			}()
		}
		for i := 0; i < len(currentNeighbors); i++ {
			aggregatedResult := <-resultChan
			next = append(next, aggregatedResult...)
		}

		// remove duplicate
		next = UniqueIDResults(next)
		for _, id := range next {
			local.AddRoute(id)
		}
		currentNeighbors = next
		level -= 1

	}

	// TODO(students): [Tapestry] Implement me!
	return nil
}

// AddNode adds node to the tapestry
//
// - Begin the acknowledged multicast
// - Return the neighborset from the multicast
func (local *TapestryNode) AddNode(
	ctx context.Context,
	nodeMsg *pb.NodeMsg,
) (*pb.Neighbors, error) {
	nodeId, err := ParseID(nodeMsg.Id)
	if err != nil {
		return nil, err
	}

	multicastRequest := &pb.MulticastRequest{
		NewNode: nodeMsg.Id,
		Level:   int32(SharedPrefixLength(nodeId, local.Id)),
	}
	return local.AddNodeMulticast(context.Background(), multicastRequest)
}

// AddNodeMulticast sends newNode to need-to-know nodes participating in the multicast.
//   - Perform multicast to need-to-know nodes
//   - Add the route for the new node (use `local.addRoute`)
//   - Transfer of appropriate router info to the new node (use `local.locationsByKey.GetTransferRegistrations`)
//     If error, rollback the location map (add back unsuccessfully transferred objects)
//
// - Propagate the multicast to the specified row in our routing table and await multicast responses
// - Return the merged neighbor set
//
// - note: `local.table.GetLevel` does not return the local node so you must manually add this to the neighbors set
func (local *TapestryNode) AddNodeMulticast(
	ctx context.Context,
	multicastRequest *pb.MulticastRequest,
) (*pb.Neighbors, error) {
	if len(multicastRequest.NewNode) == 0 {
		local.log.Println("new node empty")
	}
	local.log.Println(multicastRequest.NewNode)
	newNodeId, err := ParseID(multicastRequest.NewNode)
	if err != nil {
		return nil, err
	}
	// var mutex sync.Mutex

	level := int(multicastRequest.Level)
	// only do transfer in the last round for each node
	if level >= DIGITS {
		local.AddRoute(newNodeId)
		// transfer
		go func() {
			objects := local.LocationsByKey.GetTransferRegistrations(local.Id, newNodeId)
			conn := local.Node.PeerConns[local.RetrieveID(newNodeId)]

			NewNode := pb.NewTapestryRPCClient(conn)
			data := make(map[string]*pb.Neighbors)
			for key, nodes := range objects {
				data[key] = &pb.Neighbors{
					Neighbors: idsToStringSlice(nodes),
				}
			}

			transferData := &pb.TransferData{
				From: local.String(),
				Data: data,
			}

			_, err := NewNode.Transfer(context.Background(), transferData)
			if err != nil {
				local.RemoveBadNodes(context.Background(), &pb.Neighbors{Neighbors: []string{string(newNodeId.String())}})
			}
		}()
		return &pb.Neighbors{Neighbors: []string{}}, nil
	}

	targets := append(local.Table.GetLevel(level), local.Id)
	local.log.Printf("Add node multicast %v at level %v\n", newNodeId, level)
	result := pb.Neighbors{Neighbors: []string{}}
	resultChan := make(chan []string)

	for _, target := range targets {
		go func(target ID, level int) {
			conn := local.Node.PeerConns[local.RetrieveID(target)]
			targetNode := pb.NewTapestryRPCClient(conn)
			local.log.Println("build the conn at level:", level)

			newMulticastRequst := &pb.MulticastRequest{NewNode: multicastRequest.NewNode,
				Level: int32(level + 1)}
			// newMulticastRequst.Level += 1
			local.log.Println("in level:", newMulticastRequst.Level)
			neighbours, err := targetNode.AddNodeMulticast(context.Background(), newMulticastRequst)
			if err != nil {
				local.log.Println("AddNodeMulticast recursive call failed")
				local.RemoveBadNodes(context.Background(), &pb.Neighbors{Neighbors: []string{string(target.String())}})
			} else {
				if len(neighbours.Neighbors) > 0 {
					resultChan <- neighbours.Neighbors
				} else {
					resultChan <- []string{}
				}
			}

		}(target, level)

	}
	for i := 0; i < len(targets); i++ {
		aggregatedResult := <-resultChan
		result.Neighbors = append(result.Neighbors, aggregatedResult...)
	}

	result.Neighbors = append(result.Neighbors, idsToStringSlice(targets)...)

	// remove deplicate
	result.Neighbors = removeStringDuplicates(result.Neighbors)
	fmt.Println("len of neighbors:", len(result.Neighbors))
	return &result, nil

	// TODO(students): [Tapestry] Implement me!

}

func removeStringDuplicates(neighbors []string) []string {
	encountered := make(map[string]bool)
	result := []string{}
	for _, neighbor := range neighbors {
		if !encountered[neighbor] {
			encountered[neighbor] = true
			result = append(result, neighbor)
		}
	}
	return result
}

// AddBackpointer adds the from node to our backpointers, and possibly add the node to our
// routing table, if appropriate
func (local *TapestryNode) AddBackpointer(
	ctx context.Context,
	nodeMsg *pb.NodeMsg,
) (*pb.Ok, error) {
	id, err := ParseID(nodeMsg.Id)
	if err != nil {
		return nil, err
	}

	if local.Backpointers.Add(id) {
		local.log.Printf("Added backpointer %v\n", id)
	}
	local.AddRoute(id)

	ok := &pb.Ok{
		Ok: true,
	}
	return ok, nil
}

// RemoveBackpointer removes the from node from our backpointers
func (local *TapestryNode) RemoveBackpointer(
	ctx context.Context,
	nodeMsg *pb.NodeMsg,
) (*pb.Ok, error) {
	id, err := ParseID(nodeMsg.Id)
	if err != nil {
		return nil, err
	}

	if local.Backpointers.Remove(id) {
		local.log.Printf("Removed backpointer %v\n", id)
	}

	ok := &pb.Ok{
		Ok: true,
	}
	return ok, nil
}

// GetBackpointers gets all backpointers at the level specified, and possibly adds the node to our
// routing table, if appropriate
func (local *TapestryNode) GetBackpointers(
	ctx context.Context,
	backpointerReq *pb.BackpointerRequest,
) (*pb.Neighbors, error) {
	id, err := ParseID(backpointerReq.From)
	if err != nil {
		return nil, err
	}
	level := int(backpointerReq.Level)

	local.log.Printf("Sending level %v backpointers to %v\n", level, id)
	backpointers := local.Backpointers.Get(level)
	err = local.AddRoute(id)
	if err != nil {
		return nil, err
	}

	resp := &pb.Neighbors{
		Neighbors: idsToStringSlice(backpointers),
	}
	return resp, err
}

// RemoveBadNodes discards all the provided nodes
// - Remove each node from our routing table
// - Remove each node from our set of backpointers
func (local *TapestryNode) RemoveBadNodes(
	ctx context.Context,
	neighbors *pb.Neighbors,
) (*pb.Ok, error) {
	badnodes, err := stringSliceToIds(neighbors.Neighbors)
	if err != nil {
		return nil, err
	}

	for _, badnode := range badnodes {
		if local.Table.Remove(badnode) {
			local.log.Printf("Removed bad node %v\n", badnode)
		}
		if local.Backpointers.Remove(badnode) {
			local.log.Printf("Removed bad node backpointer %v\n", badnode)
		}
	}

	resp := &pb.Ok{
		Ok: true,
	}
	return resp, nil
}

// Utility function that adds a node to our routing table.
//
// - Adds the provided node to the routing table, if appropriate.
// - If the node was added to the routing table, notify the node of a backpointer
// - If an old node was removed from the routing table, notify the old node of a removed backpointer
func (local *TapestryNode) AddRoute(remoteNodeId ID) error {
	ok, prevId := local.Table.Add(remoteNodeId)
	if ok {
		if prevId != nil {
			go func(prevId *ID) {
				conn := local.Node.PeerConns[local.RetrieveID(*prevId)]
				prevNode := pb.NewTapestryRPCClient(conn)
				rmMsg := &pb.NodeMsg{Id: local.String()}
				_, err := prevNode.RemoveBackpointer(context.Background(), rmMsg)
				if err != nil {
					local.log.Println("remove backpointer failed")
				}
			}(prevId)

		}
		go func() {
			conn2 := local.Node.PeerConns[local.RetrieveID(remoteNodeId)]
			remoteNode := pb.NewTapestryRPCClient(conn2)
			addMsg := &pb.NodeMsg{Id: local.String()}
			_, err := remoteNode.AddBackpointer(context.Background(), addMsg)
			if err != nil {
				local.Table.Remove(remoteNodeId)
				local.Backpointers.Remove(remoteNodeId)
				local.log.Println("add backpointer failed")
			}
		}()

	}

	// TODO(students): [Tapestry] Implement me!
	return nil
}