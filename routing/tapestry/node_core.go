/*
 *  Brown University, CS138, Spring 2023
 *
 *  Purpose: Defines functions to publish and lookup objects in a Tapestry mesh
 */

package tapestry

import (
	"context"
	"errors"
	"fmt"
	pb "modist/proto"
	"time"
)

// Store a blob on the local node and publish the key to the tapestry.
func (local *TapestryNode) Store(key string, value []byte) (err error) {
	done, err := local.Publish(key)
	if err != nil {
		return err
	}
	local.blobstore.Put(key, value, done)
	return nil
}

// Get looks up a key in the tapestry then fetch the corresponding blob from the
// remote blob store.
func (local *TapestryNode) Get(key string) ([]byte, error) {
	// Lookup the key
	routerIds, err := local.Lookup(key)
	if err != nil {
		return nil, err
	}
	if len(routerIds) == 0 {
		return nil, fmt.Errorf("No routers returned for key %v", key)
	}

	// Contact router
	keyMsg := &pb.TapestryKey{
		Key: key,
	}

	var errs []error
	for _, routerId := range routerIds {
		conn := local.Node.PeerConns[local.RetrieveID(routerId)]
		router := pb.NewTapestryRPCClient(conn)
		resp, err := router.BlobStoreFetch(context.Background(), keyMsg)
		if err != nil {
			errs = append(errs, err)
		} else if resp.Data != nil {
			return resp.Data, nil
		}
	}

	return nil, fmt.Errorf("Error contacting routers, %v: %v", routerIds, errs)
}

// Remove the blob from the local blob store and stop advertising
func (local *TapestryNode) Remove(key string) bool {
	return local.blobstore.Delete(key)
}

// Publishes the key in tapestry.
//
// - Start periodically publishing the key. At each publishing:
//   - Find the root node for the key
//   - Register the local node on the root
//   - if anything failed, retry; until RETRIES has been reached.
//
// - Return a channel for cancelling the publish
//   - if receiving from the channel, stop republishing
//
// Some note about publishing behavior:
//   - The first publishing attempt should attempt to retry at most RETRIES times if there is a failure.
//     i.e. if RETRIES = 3 and FindRoot errored or returned false after all 3 times, consider this publishing
//     attempt as failed. The error returned for Publish should be the error message associated with the final
//     retry.
//   - If any of these attempts succeed, you do not need to retry.
//   - In addition to the initial publishing attempt, you should repeat this entire publishing workflow at the
//     appropriate interval. i.e. every 5 seconds we attempt to publish, and THIS publishing attempt can either
//     succeed, or fail after at most RETRIES times.
//   - Keep trying to republish regardless of how the last attempt went
func (local *TapestryNode) Publish(key string) (chan bool, error) {
	// initial publishing (can have RETRIES attempts)
	err := local.publishWithRetries(key)
	if err != nil {
		return nil, err
	}

	// republishing goroutine
	// we keep republishing so that if root node is down, data can be published to a new root node
	cancelChan := make(chan bool)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			// stop republishing goroutine if receive from cancelling channel
			// does not actually remove data or notify root node that we no longer host the data
			case <-cancelChan:
				return
			// republish every 5 seconds
			case <-ticker.C:
				// each republish attempt has RETRIES attempt
				// republish goes on regardless of last republish's result
				local.publishWithRetries(key)
			}
		}
	}()

	// return the cancelling channel
	return cancelChan, nil

	// TODO(students): [Tapestry] Implement me!
	// return nil, errors.New("Publish has not been implemented yet!")
}

// helper function for publishing key (find root and register key at root) with RETRIES
// if succeeded within RETRIES times, return nil
// otherwise return the last error encountered
func (local *TapestryNode) publishWithRetries(key string) error {
	// initial publishing (can have RETRIES attempts)
	for i := 0; i < RETRIES; i++ {
		// find root node for the key
		idMsg := &pb.IdMsg{
			Id:    Hash(key).String(),
			Level: 0,
		}
		retRootMsg, err := local.FindRoot(context.Background(), idMsg)
		if err != nil {
			// cannot find root on last retry
			// attempt is considered failed and return last error
			if i == RETRIES-1 {
				return err
			} else { // next retry
				continue
			}
		}

		// call register on the root node
		rootId, err := ParseID(retRootMsg.Next)
		if err != nil {
			// last retry, return error
			if i == RETRIES-1 {
				return err
			} else { // next retry
				continue
			}
		}
		registrationMsg := &pb.Registration{
			FromNode: local.Id.String(),
			Key:      key,
		}
		conn := local.Node.PeerConns[local.RetrieveID(rootId)]
		rootNode := pb.NewTapestryRPCClient(conn)
		retRegMsg, err := rootNode.Register(context.Background(), registrationMsg)
		if err != nil || !retRegMsg.Ok {
			// last retry, return error
			if i == RETRIES-1 {
				return err
			} else { // next retry
				continue
			}
		}
		// find root and register succeeded, no need to retry
		break
	}
	// publish succeeded, no error
	return nil
}

// Lookup look up the Tapestry nodes that are storing the blob for the specified key.
//
// - Find the root node for the key
// - Fetch the routers (nodes storing the blob) from the root's location map
// - Attempt up to RETRIES times
func (local *TapestryNode) Lookup(key string) ([]ID, error) {
	// find root node

	// call fetch RPC on the root to get the nodes ID that store key
	// errors: node is not root; node ID list is empty
	var retFetchMsg *pb.FetchedLocations
	for i := 0; i < RETRIES; i++ {
		// find root node for the key
		idMsg := &pb.IdMsg{
			Id:    Hash(key).String(),
			Level: 0,
		}
		retRootMsg, err := local.FindRoot(context.Background(), idMsg)
		if err != nil {
			// cannot find root on last retry
			// attempt is considered failed and return last error
			if i == RETRIES-1 {
				return nil, err
			} else { // next retry
				continue
			}
		}

		// call fetch RPC on the root to get the nodes ID that store key
		// errors: node is not root; node ID list is empty
		rootId, err := ParseID(retRootMsg.Next)
		if err != nil {
			// last retry, return error
			if i == RETRIES-1 {
				return nil, err
			} else { // next retry
				continue
			}
		}
		keyMsg := &pb.TapestryKey{
			Key: key,
		}
		conn := local.Node.PeerConns[local.RetrieveID(rootId)]
		rootNode := pb.NewTapestryRPCClient(conn)
		retFetchMsg, err = rootNode.Fetch(context.Background(), keyMsg)
		if err != nil || !retFetchMsg.IsRoot {
			// last retry, return error
			if i == RETRIES-1 {
				return nil, err
			} else { // next retry
				continue
			}
		}
		// find root and fetch succeeded, no need to retry
		break
	}
	ids, err := stringSliceToIds(retFetchMsg.Values) // can be empty; handled by Get()
	return ids, err

	// TODO(students): [Tapestry] Implement me!
	// return nil, errors.New("Lookup has not been implemented yet!")
}

// FindRoot returns the root for the id in idMsg by recursive RPC calls on the next hop found in our routing table
//   - find the next hop from our routing table
//   - call FindRoot on nextHop
//   - if failed, add nextHop to toRemove, remove them from local routing table, retry
func (local *TapestryNode) FindRoot(ctx context.Context, idMsg *pb.IdMsg) (*pb.RootMsg, error) {
	id, err := ParseID(idMsg.Id)
	if err != nil {
		return nil, err
	}
	level := idMsg.Level

	failedNodes := []string{}
	var retRootMsg *pb.RootMsg
	for { // loop to keep contacting new next hop if we encounter error
		nextHopId := local.Table.FindNextHop(id, level)
		// FindNextHop will return local node's ID if we are at the root
		if nextHopId.String() == local.Id.String() {
			return &pb.RootMsg{
				Next:     local.Id.String(),
				ToRemove: []string{},
			}, nil
		}

		// call FindRoot on nextHop
		newIdMsg := &pb.IdMsg{
			Id:    idMsg.Id,
			Level: level + 1,
		}
		conn := local.Node.PeerConns[local.RetrieveID(nextHopId)]
		nextHopNode := pb.NewTapestryRPCClient(conn)
		retRootMsg, err = nextHopNode.FindRoot(ctx, newIdMsg)

		if err == nil {
			break
		} // next hop successfully contacted
		// handle timeout or other error on FindRoot to nextHop:
		// remove next hop from local routing table
		local.Table.Remove(nextHopId)
		local.Backpointers.Remove(nextHopId)
		// add next hop to a set of nodes to be removed
		failedNodes = append(failedNodes, nextHopId.String())
	}

	// remove ToRemove nodes from the RootMsg we got from downstream nodes
	toRemoveNodeIds, _ := stringSliceToIds(retRootMsg.ToRemove)
	for _, id := range toRemoveNodeIds {
		local.Table.Remove(id)
		local.Backpointers.Remove(id) // remove can handle case where id is not in table
	}

	// return to upstream nodes with new nodes to be removed added
	return &pb.RootMsg{
		Next:     retRootMsg.Next,
		ToRemove: append(retRootMsg.ToRemove, failedNodes...),
	}, nil

	// TODO(students): [Tapestry] Implement me!
}

// The node that stores some data with key is registering themselves to us as an advertiser of the key.
// - Check that we are the root node for the key, return true in pb.Ok if we are
// - Add the node to the location map (local.locationsByKey.Register)
//   - local.locationsByKey.Register kicks off a timer to remove the node if it's not advertised again
//     after TIMEOUT
func (local *TapestryNode) Register(
	ctx context.Context,
	registration *pb.Registration,
) (*pb.Ok, error) {
	from, err := ParseID(registration.FromNode)
	if err != nil {
		return nil, err
	}
	key := registration.Key

	// Check that we are the root node for the key, return false if not
	idMsg := &pb.IdMsg{
		Id:    Hash(key).String(),
		Level: 0,
	}
	retRootMsg, err := local.FindRoot(ctx, idMsg)
	if err != nil || retRootMsg.Next != local.Id.String() {
		return &pb.Ok{
			Ok: false,
		}, err
	}

	// Add node to location map
	// the following fn already handles removing node from locationsMap if it stops re-publishing after TIMEOUT
	local.LocationsByKey.Register(key, from, TIMEOUT)

	return &pb.Ok{
		Ok: true,
	}, nil

	// TODO(students): [Tapestry] Implement me!
}

// Fetch checks that we are the root node for the requested key and
// return all nodes that are registered in the local location map for this key
func (local *TapestryNode) Fetch(
	ctx context.Context,
	key *pb.TapestryKey,
) (*pb.FetchedLocations, error) {
	// Check that we are the root node for the key, return immediately if not
	keyStr := key.GetKey()
	idMsg := &pb.IdMsg{
		Id:    Hash(keyStr).String(),
		Level: 0,
	}
	retRootMsg, err := local.FindRoot(ctx, idMsg)
	if err != nil || retRootMsg.Next != local.Id.String() {
		return &pb.FetchedLocations{
			IsRoot: false,
			Values: nil,
		}, err
	}

	// return all nodes in location map for this key
	nodeIds := local.LocationsByKey.Get(keyStr)
	return &pb.FetchedLocations{
		IsRoot: true,
		Values: idsToStringSlice(nodeIds),
	}, nil
	// return nil, errors.New("Fetch not implemented yet")
	// TODO(students): [Tapestry] Implement me!
}

// Retrieves the blob corresponding to a key
func (local *TapestryNode) BlobStoreFetch(
	ctx context.Context,
	key *pb.TapestryKey,
) (*pb.DataBlob, error) {
	data, isOk := local.blobstore.Get(key.Key)

	var err error
	if !isOk {
		err = errors.New("Key not found")
	}

	return &pb.DataBlob{
		Key:  key.Key,
		Data: data,
	}, err
}

// Transfer registers all of the provided objects in the local location map. (local.locationsByKey.RegisterAll)
// If appropriate, add the from node to our local routing table
func (local *TapestryNode) Transfer(
	ctx context.Context,
	transferData *pb.TransferData,
) (*pb.Ok, error) {
	from, err := ParseID(transferData.From)
	if err != nil {
		return &pb.Ok{Ok: false}, nil
	}

	nodeMap := make(map[string][]ID)
	for key, set := range transferData.Data {
		nodeMap[key], err = stringSliceToIds(set.Neighbors)
		if err != nil {
			return &pb.Ok{Ok: false}, nil
		}
	}
	local.LocationsByKey.RegisterAll(nodeMap, TIMEOUT)
	local.AddRoute(from)

	// TODO(students): [Tapestry] Implement me!
	return &pb.Ok{Ok: true}, nil
}

// calls FindRoot on a remote node to find the root of the given id
func (local *TapestryNode) FindRootOnRemoteNode(remoteNodeId ID, id ID) (*ID, error) {
	// TODO(students): [Tapestry] Implement me!
	conn := local.Node.PeerConns[local.RetrieveID(remoteNodeId)]
	remoteNode := pb.NewTapestryRPCClient(conn)

	rootMsg, err := remoteNode.FindRoot(context.Background(), &pb.IdMsg{
		Id:    id.String(),
		Level: 0,
	})
	if err != nil {
		local.Table.Remove(remoteNodeId)
		local.Backpointers.Remove(remoteNodeId)
		local.log.Println("error findroot in findRemoteNode")
		return nil, err
	}
	root, err := ParseID(rootMsg.Next)
	if err != nil {
		local.log.Println("error Parsing rootId")
		return nil, err
	}

	return &root, nil
}