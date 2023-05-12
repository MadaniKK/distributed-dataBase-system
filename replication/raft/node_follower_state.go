package raft

import (
	"context"
	"math"
	"math/rand"
	pb "modist/proto"
	"time"
)

// doFollower implements the logic for a Raft node in the follower state.
func (rn *RaftNode) doFollower() stateFunction {
	rn.state = FollowerState
	rn.log.Printf("transitioning to %s state at term %d", rn.state, rn.GetCurrentTerm())

	// TODO(students): [Raft] Implement me!
	// Hint: perform any initial work, and then consider what a node in the
	// follower state should do when it receives an incoming message on every
	// possible channel.

	// currently all goroutines in this file does not hear from cancel channel
	// because they are short lived and does not keep running in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// set timeout
	// @@re-randomize election timeout every term?
	rand.Seed(time.Now().UnixNano())
	minTimeout := rn.electionTimeout
	maxTimeout := 2 * rn.electionTimeout
	randomElectionTO := time.Duration(rand.Int63n(int64(maxTimeout-minTimeout+1)) + int64(minTimeout))
	ticker := time.NewTicker(randomElectionTO)
	defer ticker.Stop()

	rn.setVotedFor(0)

	// channels respond
	// AppendEntries:
	// commit check (start and end): update commitIndex and lastApplied, update log => if index and term matches
	// reject if does not find entry with same index and term as prevLogIndex and prevLogTerm

	// propose: forward request to leader; stop node if channel is closed (#445); drop if leader is unknown

	// requestvotes: check up-to-dateness

	// timeout: increment term, start election (transition to candidate state)

	// cancel: end goroutine
	for {
		select {
		case msg := <-rn.appendEntriesC:
			rn.checkCommit()
			rn.handleAppendEntries(msg, ticker, randomElectionTO)
			rn.checkCommit()
		case data, ok := <-rn.proposeC:
			// stop node if proposeC is closed
			if !ok {
				// close(rn.stopC)
				// rn.Stop()
				rn.proposeC = nil
				rn.state = ExitState
				rn.Stop()
				close(rn.stopC)
				return nil
			}
			rn.handlePropose(data, ctx)
		case msg := <-rn.requestVoteC:
			rn.handleRequestVote(msg, ticker, randomElectionTO)
		case <-ticker.C:
			return rn.doCandidate
		}
	}
}

func (rn *RaftNode) prevEntryMatch(prevLogIndex uint64, prevLogTerm uint64) bool {
	// no prevLogIndex entry exists
	logEntry := rn.GetLog(prevLogIndex)
	if logEntry == nil {
		return false
	}
	return logEntry.GetTerm() == prevLogTerm
}

func (rn *RaftNode) handleAppendEntries(msg AppendEntriesMsg, ticker *time.Ticker, randomElectionTO time.Duration) {
	// reset election timeout
	ticker.Reset(randomElectionTO)

	request := msg.request
	// failure cases
	if request.Term < rn.GetCurrentTerm() || !rn.prevEntryMatch(request.PrevLogIndex, request.PrevLogTerm) {
		go func() {
			msg.reply <- pb.AppendEntriesReply{
				From:    rn.node.ID,
				To:      request.From,
				Term:    rn.GetCurrentTerm(),
				Success: false,
			}
		}()
		return
	}

	// success case (follower and leader log matches on prevLogIndex)
	// update log: truncate [prevLogIndex + 1, end], append leader's log
	rn.TruncateLog(request.PrevLogIndex + 1)
	for _, entry := range request.Entries {
		rn.StoreLog(entry)
	}
	// update states
	rn.leader = request.From
	if request.Term > rn.GetCurrentTerm() {
		rn.SetCurrentTerm(request.Term)
		rn.setVotedFor(0) // reset votedFor for this new term
	}

	// update commitIdx
	if request.LeaderCommit > rn.commitIndex {
		rn.commitIndex = uint64(math.Min(float64(rn.LastLogIndex()), float64(request.LeaderCommit)))
	}

	// send reply
	go func() {
		msg.reply <- pb.AppendEntriesReply{
			From:    rn.node.ID,
			To:      request.From,
			Term:    rn.GetCurrentTerm(),
			Success: true,
		}
	}()
}

// send [lastApplied + 1, commitIndex] to commitC
// update lastApplied to commitIndex
// if entry is NOOP do not send to commitC
func (rn *RaftNode) checkCommit() {
	for rn.commitIndex > rn.lastApplied {
		rn.lastApplied++
		logEntry := rn.GetLog(rn.lastApplied)
		if logEntry.Data != nil {
			go func() {
				data := commit(logEntry.GetData())
				rn.commitC <- &data
			}()
		}
	}
}

func (rn *RaftNode) handlePropose(data []byte, ctx context.Context) {
	// drop if leader is unknown
	if rn.leader == 0 {
		return
	}
	// forward to leader
	go func() {
		conn := rn.node.PeerConns[uint64(rn.leader)]
		leaderNode := pb.NewRaftRPCClient(conn)
		proposeRequest := &pb.ProposalRequest{
			From: rn.node.ID,
			To:   rn.leader,
			Data: data,
		}
		// propose reply is empty so no need to do anything
		_, err := leaderNode.Propose(ctx, proposeRequest)
		if err != nil {
			rn.log.Printf("Error when forwarding request to leader %v from node %v", rn.leader, rn.node.ID)
		}
	}()
}

func (rn *RaftNode) candidateUpToDate(cLastLogIndex uint64, cLastLogTerm uint64) bool {
	fLastLogIndex := rn.LastLogIndex()
	if fLastLogIndex == 0 {
		return true // we don't have log entries yet
	}
	fLastLogTerm := rn.GetLog(fLastLogIndex).GetTerm()
	if fLastLogTerm > cLastLogTerm {
		return false // we have later term
	}
	if fLastLogTerm == cLastLogTerm && fLastLogIndex > cLastLogIndex {
		return false // we have same term but longer log
	}
	return true
}

func (rn *RaftNode) handleRequestVote(msg RequestVoteMsg, ticker *time.Ticker, randomElectionTO time.Duration) {
	request := msg.request
	// failure cases:
	// candidate has smaller term || already voted for another node in this term || candidate log is not as up-to-date as ours
	if request.Term < rn.GetCurrentTerm() || (rn.GetVotedFor() != 0 && rn.GetVotedFor() != request.From) || !rn.candidateUpToDate(request.LastLogIndex, request.LastLogTerm) {
		go func() {
			msg.reply <- pb.RequestVoteReply{
				From:        rn.node.ID,
				To:          request.From,
				Term:        rn.GetCurrentTerm(),
				VoteGranted: false,
			}
		}()
		return
	}

	// success case, grant vote
	go func() {
		msg.reply <- pb.RequestVoteReply{
			From:        rn.node.ID,
			To:          request.From,
			Term:        request.Term,
			VoteGranted: true,
		}
	}()
	// reset election timeout
	ticker.Reset(randomElectionTO)

	// update states
	rn.SetCurrentTerm(request.Term)
	rn.setVotedFor(request.From)
}
