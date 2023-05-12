package raft

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"modist/orchestrator/node"
	pb "modist/proto"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RETRIES is the number of retries upon failure. By default, we have 3
const RETRIES = 3

// RETRY_TIME is the amount of time to wait between retries
const RETRY_TIME = 100 * time.Millisecond

type State struct {
	// In-memory key value store
	store map[string]string
	mu    sync.RWMutex

	// Channels given back by the underlying Raft implementation
	proposeC  chan<- []byte
	commitC   <-chan *commit
	dataChans map[uuid.UUID](chan []byte)
	// dataC chan *pb.ResolvableKV

	// Observability
	log *log.Logger

	// The public-facing API that this replicator must implement
	pb.ReplicatorServer
}

type Args struct {
	Node   *node.Node
	Config *Config
}

// Configure is called by the orchestrator to start this node
//
// The "args" are any to support any replicator that might need arbitrary
// set of configuration values.
//
// You are free to add any fields to the State struct that you feel may help
// with implementing the ReplicateKey or GetReplicatedKey functions. Likewise,
// you may call any additional functions following the second TODO. Apart
// from the two TODOs, DO NOT EDIT any other part of this function.
func Configure(args any) *State {
	a := args.(Args)

	node := a.Node
	config := a.Config

	proposeC := make(chan []byte)
	commitC := NewRaftNode(node, config, proposeC)
	dataChans := make(map[uuid.UUID]chan []byte)

	// dataC := make(chan *pb.ResolvableKV)

	s := &State{
		store:     make(map[string]string),
		proposeC:  proposeC,
		commitC:   commitC,
		dataChans: dataChans,

		log: node.Log,

		// TODO(students): [Raft] Initialize any additional fields and add to State struct
	}

	// We registered RaftRPCServer when calling NewRaftNode above so we only need to
	// register ReplicatorServer here
	s.log.Printf("starting gRPC server at %s", node.Addr.Host)
	grpcServer := node.GrpcServer
	pb.RegisterReplicatorServer(grpcServer, s)
	go grpcServer.Serve(node.Listener)
	go func() {
		for data := range commitC {
			var kv *KV
			err := json.Unmarshal(*data, kv)
			if err != nil {
				s.log.Printf("error Unmarshalling %s", err)
				continue
			}
			s.mu.Lock()
			s.store[kv.Key] = kv.Value
			s.dataChans[kv.uuid] <- *data
			s.mu.Unlock()
		}
	}()

	// TODO(students): [Raft] Call helper functions if needed

	return s
}

type KV struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	uuid  uuid.UUID
}

// ReplicateKey replicates the (key, value) given in the PutRequest by relaying it to the
// underlying Raft cluster/implementation.
//
// You should first marshal the KV struct using the encoding/json library. Then, put the
// marshalled struct in the proposeC channel and wait until the Raft cluster has confirmed that
// it has been committed. Think about how you can use the commitC channel to find out when
// something has been committed.
//
// If the proposal has not been committed after RETRY_TIME, you should retry it. You should retry
// the proposal a maximum of RETRIES times, at which point you can return a nil reply and an error.
func (s *State) ReplicateKey(ctx context.Context, r *pb.PutRequest) (*pb.PutReply, error) {

	// f := func()(*pb.PutReply, error){

	// 	return nil, nil

	// }
	uuid := uuid.New()
	proposal := &KV{Key: r.Key, Value: r.Value, uuid: uuid}
	data, err := json.Marshal(proposal)
	if err != nil {
		return nil, nil
	}
	s.mu.Lock()
	dataC := make(chan []byte)
	s.dataChans[uuid] = dataC
	s.mu.Unlock()
	proposeTimer := time.NewTicker(RETRY_TIME)
	s.proposeC <- data
	retry := 0
	var reply *pb.PutReply
	for {
		select {
		case <-proposeTimer.C:
			if retry == RETRIES {
				if reply == nil {
					return nil, errors.New("replicate failed")
				}
			}
			retry++
			s.proposeC <- data
			proposeTimer.Reset(RETRY_TIME)
		case data := <-dataC:
			if data != nil {
				return &pb.PutReply{Clock: nil}, nil
			}

		}

	}

	// // TODO(students): [Raft] Implement me!
}

// GetReplicatedKey reads the given key from s.store. The implementation of
// this method should be pretty short (roughly 8-12 lines).
func (s *State) GetReplicatedKey(ctx context.Context, r *pb.GetRequest) (*pb.GetReply, error) {
	key := r.GetKey()
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.store[key]
	if ok {
		reply := &pb.GetReply{
			Value: value,
			Clock: nil,
		}
		return reply, nil
	} else {
		s.log.Println("cannot find the key")
		return nil, nil
	}

	// TODO(students): [Raft] Implement me!

}
