package raft

import (
	"context"
	"math"
	"modist/orchestrator/node"
	"modist/proto"
	"time"
)

// doLeader implements the logic for a Raft node in the leader state.
func (rn *RaftNode) doLeader() stateFunction {
	rn.log.Printf("transitioning to leader state at term %d", rn.GetCurrentTerm())
	rn.state = LeaderState

	rn.leader = rn.node.ID
	rn.setVotedFor(rn.node.ID)

	// Upon election: add log to self, send AppendEntries
	rn.StoreLog(&proto.LogEntry{
		Index: rn.LastLogIndex() + 1,
		Term:  rn.GetCurrentTerm(),
		Type:  proto.EntryType_NORMAL,
		Data:  nil,
	})
	rn.leaderMu.Lock()
	for _, peer := range rn.node.PeerNodes {
		if peer.ID == rn.node.ID {
			rn.nextIndex[peer.ID] = rn.LastLogIndex() + 1
			rn.matchIndex[peer.ID] = rn.LastLogIndex()

		} else {
			rn.nextIndex[peer.ID] = rn.LastLogIndex()
			rn.matchIndex[peer.ID] = uint64(0)
		}
	}
	rn.leaderMu.Unlock()
	heartbeatTimer := time.NewTicker(rn.heartbeatTimeout)
	defer heartbeatTimer.Stop()
	// time.Sleep(rn.heartbeatTimeout)
	stepDown := make(chan bool, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		toFollower := rn.sendHeartbeats(ctx)
		if toFollower {
			stepDown <- true
		}

	}()

	for {
		select {
		case <-stepDown:
			rn.state = FollowerState
			return rn.doFollower
		// case <-rn.stopC:
		// 	rn.Stop()
		// 	return nil
		case <-heartbeatTimer.C:
			go func() {
				isFollowerNow := rn.sendHeartbeats(ctx)
				if isFollowerNow {
					stepDown <- true
				}
			}()
			heartbeatTimer.Reset(rn.heartbeatTimeout)

		case data, ok := <-rn.proposeC:
			if !ok {
				rn.proposeC = nil
				rn.state = ExitState
				rn.Stop()
				close(rn.stopC)
				return nil

			}
			if data != nil {
				// handle client request
				// append entry to local log
				rn.StoreLog(&proto.LogEntry{
					Index: rn.LastLogIndex() + 1,
					Term:  rn.GetCurrentTerm(),
					Type:  proto.EntryType_NORMAL,
					Data:  data,
				})
				// rn.matchIndex[rn.node.ID] = rn.LastLogIndex()
				// rn.nextIndex[rn.node.ID] = rn.LastLogIndex() + 1
				go func() {
					isFollowerNow := rn.sendHeartbeats(ctx)
					if isFollowerNow {
						stepDown <- true
					}
				}()
				heartbeatTimer.Reset(rn.heartbeatTimeout)
			}

		// other candidate send vote request, handle vote request
		case msg := <-rn.requestVoteC:
			toFollower := rn.handleRequestVote2(msg)
			if toFollower {
				return rn.doFollower
			}
		// case RequestVoteMsg := <-rn.requestVoteC:
		// 	req := RequestVoteMsg.request
		// 	replyC := RequestVoteMsg.reply
		// 	if req.Term < rn.GetCurrentTerm() {
		// 		replyC <- proto.RequestVoteReply{
		// 			From:        rn.node.ID,
		// 			To:          req.From,
		// 			Term:        rn.GetCurrentTerm(),
		// 			VoteGranted: false,
		// 		}
		// 	}
		// 	if req.Term > rn.GetCurrentTerm() {
		// 		// update the term no matter grant vote or not
		// 		rn.SetCurrentTerm(req.Term)
		// 		//reset the vote for new term
		// 		rn.setVotedFor(uint64(0))
		// 	}
		// 	// check if the candiate's log is upToDate
		// 	voteFor := rn.GetVotedFor() == uint64(0) || rn.GetVotedFor() == req.From
		// 	var upToDate bool
		// 	logIndx := rn.LastLogIndex()
		// 	if req.LastLogTerm > rn.GetLog(logIndx).Term {
		// 		upToDate = true
		// 	} else if req.LastLogTerm == rn.GetLog(logIndx).Term {
		// 		upToDate = req.LastLogIndex >= logIndx
		// 	} else {
		// 		upToDate = false
		// 	}

		// 	voteGranted := voteFor && upToDate
		// 	replyC <- proto.RequestVoteReply{
		// 		From:        rn.node.ID,
		// 		To:          req.From,
		// 		Term:        rn.GetCurrentTerm(),
		// 		VoteGranted: voteGranted,
		// 	}
		// 	if rn.GetVotedFor() == uint64(0) || voteGranted {
		// 		rn.setVotedFor(req.From)
		// 	}
		// 	if voteGranted {
		// 		rn.state = FollowerState
		// 		return rn.doFollower
		// 	}

		// recieve appendEntries
		case msg := <-rn.appendEntriesC:
			toFollower := rn.handleAppendEntries2(msg)
			if toFollower {
				return rn.doFollower
			}

		}
	}

	// TODO(students): [Raft] Implement me!
	// Hint: perform any initial work, and then consider what a node in the
	// leader state should do when it receives an incoming message on every
	// possible channel.
	// return nil
}
func (rn *RaftNode) handleAppendEntries2(msg AppendEntriesMsg) (stepDown bool) {
	req := msg.request
	appendEntriesC := msg.reply
	if req.Term < rn.GetCurrentTerm() {
		//reject the request and send back reply
		appendEntriesC <- proto.AppendEntriesReply{
			From:    rn.node.ID,
			To:      req.From,
			Term:    rn.GetCurrentTerm(),
			Success: false,
		}
		return false

	}
	if req.Term >= rn.GetCurrentTerm() {
		stepDown = true
		//update its own term and other fields
		rn.SetCurrentTerm(req.Term)
		rn.setVotedFor(0)

		// electionTimer.Reset(rn.electionTimeout)
		// handle logs locally before steping down
		prevLogIndex := req.PrevLogIndex
		var consistentLog bool

		///
		if rn.LastLogIndex() >= prevLogIndex {
			//check if the term match
			consistentLog = req.PrevLogTerm == rn.GetLog(prevLogIndex).GetTerm()
		} else {
			consistentLog = false
		}

		if !consistentLog {
			appendEntriesC <- proto.AppendEntriesReply{
				From:    rn.node.ID,
				To:      req.From,
				Term:    rn.GetCurrentTerm(),
				Success: false,
			}
		} else {
			// consistent log, so append, truncate if conflicted
			rn.TruncateLog(prevLogIndex + 1)

			for _, entry := range req.GetEntries() {
				rn.StoreLog(entry)

			}
			if req.GetLeaderCommit() > rn.commitIndex {
				//update commit index
				rn.commitIndex = uint64(math.Min(float64(req.GetLeaderCommit()), float64(rn.LastLogIndex())))

			}

			appendEntriesC <- proto.AppendEntriesReply{
				From:    rn.node.ID,
				To:      req.From,
				Term:    rn.GetCurrentTerm(),
				Success: true,
			}

		}

	}
	rn.state = FollowerState
	rn.leader = req.From
	return

}
func (rn *RaftNode) sendHeartbeats(ctx context.Context) (stepDown bool) {
	appendEntriesResponseC := make(chan *proto.AppendEntriesReply, 10)
	rn.checkAndCommit()
	// sending all other nodes appendEntriesRPC
	for _, peer := range rn.node.PeerNodes {
		if peer.ID == rn.node.ID {
			continue
		}

		go func(peer *node.Node) {
			var entries []*proto.LogEntry

			for i := rn.nextIndex[peer.ID]; i <= rn.LastLogIndex(); i++ {
				entries = append(entries, rn.GetLog(i))
			}

			conn := rn.node.PeerConns[uint64(peer.ID)]
			raftRPCClient := proto.NewRaftRPCClient(conn)

			// the immediately precedding the new ones
			prevIndex := rn.nextIndex[peer.ID]
			if prevIndex > 0 { // or >1?
				prevIndex--
			}

			appendEntriesRequest := &proto.AppendEntriesRequest{
				From:         rn.node.ID,
				To:           peer.ID,
				Term:         rn.GetCurrentTerm(),
				PrevLogIndex: prevIndex,
				PrevLogTerm:  rn.GetLog(prevIndex).GetTerm(),
				Entries:      entries,
				LeaderCommit: rn.commitIndex,
			}
			appendReply, err := raftRPCClient.AppendEntries(ctx, appendEntriesRequest)
			if err != nil {
				rn.log.Printf("error requesting appendEntries reply from peer %d: %v", peer.ID, err)
				return
			}
			appendEntriesResponseC <- appendReply
		}(peer)
	}
	for i := 0; i < len(rn.node.PeerNodes)-1; i++ {
		select {

		case appendReply := <-appendEntriesResponseC:
			rn.log.Printf("leader commitindex before :%d", rn.commitIndex)
			if appendReply.GetTerm() > rn.GetCurrentTerm() {
				rn.SetCurrentTerm(appendReply.GetTerm())
				rn.setVotedFor(0)
				rn.state = FollowerState
				return true
			}
			// if peer accepts our appendEntries
			if appendReply.GetSuccess() {
				// update the volatile states
				rn.leaderMu.Lock()
				rn.matchIndex[appendReply.From] = rn.LastLogIndex()
				rn.nextIndex[appendReply.From] = rn.LastLogIndex() + 1
				rn.leaderMu.Unlock()
				//update urself?
				rn.matchIndex[rn.node.ID] = rn.LastLogIndex()
				rn.checkAndCommit()
				rn.log.Printf("leader commitindex :%d", rn.commitIndex)

			} else {
				// decrement the nextIndex
				rn.leaderMu.Lock()
				if rn.nextIndex[appendReply.From] > 1 {
					rn.nextIndex[appendReply.From]--
				}
				rn.leaderMu.Unlock()
			}

		}

	}

	return false
}

func (rn *RaftNode) checkAndCommit() {
	N := uint64(0)

	countNeeded := len(rn.node.PeerNodes)/2 + 1
	// >=
	for j := rn.LastLogIndex(); j > rn.commitIndex; j-- {
		count := 0
		for _, ele := range rn.matchIndex {
			if ele >= j {
				count++
			}
		}
		if count >= countNeeded && rn.GetLog(j).GetTerm() == rn.GetCurrentTerm() {
			N = j
			break
		}
	}
	rn.log.Printf("N :%d", N)
	// commit to the commitC
	for N > rn.lastApplied {
		newcommit := commit(rn.GetLog(rn.lastApplied + 1).GetData())
		go func() {
			if newcommit != nil {
				rn.commitC <- &newcommit
			}
		}()
		rn.lastApplied++

	}
	if N > rn.commitIndex {
		rn.commitIndex = N
	}

}
