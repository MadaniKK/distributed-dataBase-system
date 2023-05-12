package raft

import (
	"context"
	"math/rand"
	"modist/proto"
	"time"
)

// doCandidate implements the logic for a Raft node in the candidate state.
func (rn *RaftNode) doCandidate() stateFunction {
	rn.state = CandidateState
	rn.log.Printf("transitioning to %s state at term %d", rn.state, rn.GetCurrentTerm())
	// candidates increment its own term
	curTerm := rn.GetCurrentTerm()
	rn.SetCurrentTerm(curTerm + 1)
	//Vote for itself
	rn.setVotedFor(rn.node.ID)
	//reset election timer
	rand.Seed(time.Now().UnixNano())
	minTimeout := rn.electionTimeout
	maxTimeout := 2 * rn.electionTimeout
	electionTimeout := time.Duration(rand.Int63n(int64(maxTimeout-minTimeout+1)) + int64(minTimeout))
	// rn.electionTimeout = electionTimeout

	// construct the requestMsg

	lastLogIndex := rn.LastLogIndex()
	// if lastLogIndex == 0 {
	// 	rn.log.Println("Error, no log")
	// 	return nil
	// }
	lastLogTerm := rn.GetLog(lastLogIndex).Term

	electionTimer := time.NewTimer(electionTimeout)
	defer electionTimer.Stop()

	// channel to receive vote responses
	// reply   chan pb.RequestVoteReply
	voteResponseC := make(chan *proto.RequestVoteReply, 10)

	voteCount := 1
	votesNeeded := len(rn.node.PeerNodes)/2 + 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// send requestVotes via RPC
	for _, peer := range rn.node.PeerNodes {
		if peer.ID == rn.node.ID {
			continue
		}
		// form conn
		go func() {
			conn := rn.node.PeerConns[uint64(peer.ID)]
			raftRPCClient := proto.NewRaftRPCClient(conn)

			voteRequest := &proto.RequestVoteRequest{
				From:         rn.node.ID,
				To:           peer.ID,
				Term:         rn.GetCurrentTerm(),
				LastLogIndex: lastLogIndex,
				LastLogTerm:  lastLogTerm,
			}
			voteReply, err := raftRPCClient.RequestVote(ctx, voteRequest)
			if err != nil {
				rn.log.Printf("error requesting vote from peer %d: %v", peer.ID, err)
				return
			}
			voteResponseC <- voteReply

		}()

		for {
			select {
			// case <-rn.stopC:
			// 	rn.Stop()
			// 	return nil
			case _, ok := <-rn.proposeC:
				// check
				if !ok {
					rn.proposeC = nil
					rn.state = ExitState
					rn.Stop()
					close(rn.stopC)
					return nil
				}
			case <-electionTimer.C:
				rn.log.Printf("election timeout, starting new election")
				return rn.doCandidate

			case msg := <-rn.appendEntriesC:
				toFollower := rn.handleAppendEntries2(msg)
				if toFollower {
					return rn.doFollower
				}
			// case appendEntriesMsg := <-rn.appendEntriesC:
			// 	req := appendEntriesMsg.request
			// 	appendEntriesC := appendEntriesMsg.reply
			// 	if req.Term >= rn.GetCurrentTerm() {
			// 		//update own term and other fields
			// 		rn.SetCurrentTerm(req.Term)
			// 		rn.setVotedFor(0)
			// 		// handle logs locally before steping down
			// 		prevLogIndex := req.PrevLogIndex
			// 		var consistentLog bool

			// 		if rn.LastLogIndex() >= prevLogIndex {
			// 			//check if the term match
			// 			consistentLog = req.PrevLogTerm == rn.GetLog(prevLogIndex).GetTerm()
			// 		} else {
			// 			consistentLog = false
			// 		}
			// 		// consistentLog = rn.prevEntryMatch(req.PrevLogIndex, req.PrevLogTerm)

			// 		if !consistentLog {
			// 			appendEntriesC <- proto.AppendEntriesReply{
			// 				From:    rn.node.ID,
			// 				To:      req.From,
			// 				Term:    rn.GetCurrentTerm(),
			// 				Success: false,
			// 			}
			// 		} else {
			// 			// consistent log, so append, truncate if conflicted
			// 			rn.TruncateLog(prevLogIndex + 1)

			// 			for _, entry := range req.GetEntries() {
			// 				rn.StoreLog(entry)
			// 			}

			// 			if req.GetLeaderCommit() > rn.commitIndex {
			// 				rn.commitIndex = uint64(math.Min(float64(req.LeaderCommit), float64(rn.LastLogIndex())))

			// 			}
			// 			appendEntriesC <- proto.AppendEntriesReply{
			// 				From:    rn.node.ID,
			// 				To:      req.From,
			// 				Term:    rn.GetCurrentTerm(),
			// 				Success: true,
			// 			}
			// 			rn.leader = req.From
			// 			return rn.doFollower
			// 		}

			// 	} else {
			// 		//reject the request and send back reply
			// 		appendEntriesC <- proto.AppendEntriesReply{
			// 			From:    rn.node.ID,
			// 			To:      req.From,
			// 			Term:    rn.GetCurrentTerm(),
			// 			Success: false,
			// 		}

			// 	}

			case msg := <-rn.requestVoteC:
				toFollower := rn.handleRequestVote2(msg)
				if toFollower {
					return rn.doFollower
				}

			case voteReply := <-voteResponseC:
				if voteReply.Term > rn.GetCurrentTerm() {
					rn.state = FollowerState
					rn.SetCurrentTerm(voteReply.Term)
					rn.setVotedFor(0)
					return rn.doFollower
				}
				if voteReply.GetVoteGranted() {
					voteCount++
					if voteCount >= votesNeeded {
						rn.log.Printf("received enough votes, converting to leader")
						rn.state = LeaderState
						return rn.doLeader
					}

				}

			}

		}

	}

	// TODO(students): [Raft] Implement me!
	// Hint: perform any initial work, and then consider what a node in the
	// candidate state should do when it receives an incoming message on every
	// possible channel.
	return nil
}

func (rn *RaftNode) handleRequestVote2(msg RequestVoteMsg) (stepDown bool) {
	req := msg.request
	replyC := msg.reply
	if req.Term < rn.GetCurrentTerm() {
		replyC <- proto.RequestVoteReply{
			From:        rn.node.ID,
			To:          req.From,
			Term:        rn.GetCurrentTerm(),
			VoteGranted: false,
		}
		return false
	}
	if req.Term > rn.GetCurrentTerm() {
		// update the term no matter grant vote or not
		rn.SetCurrentTerm(req.Term)
		//reset the vote for new term
		rn.setVotedFor(uint64(0))
	}
	// check if the candiate's log is upToDate
	voteFor := rn.GetVotedFor() == uint64(0) || rn.GetVotedFor() == req.From
	var upToDate bool
	logIndx := rn.LastLogIndex()
	if req.LastLogTerm > rn.GetLog(logIndx).Term {
		upToDate = true
	} else if req.LastLogTerm == rn.GetLog(logIndx).Term {
		upToDate = req.LastLogIndex >= logIndx
	} else {
		upToDate = false
	}

	voteGranted := voteFor && upToDate
	replyC <- proto.RequestVoteReply{
		From:        rn.node.ID,
		To:          req.From,
		Term:        rn.GetCurrentTerm(),
		VoteGranted: voteGranted,
	}

	if voteGranted {
		rn.state = FollowerState
		rn.setVotedFor(req.From)
		return true
	} else {
		rn.setVotedFor(0)
	}
	return true
}
