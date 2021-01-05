package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"math/rand"
	"strconv"
	"sync"
	"time"
)
import "labrpc"

// import "bytes"
// import "labgob"


var CANDIDATE string = "Candidate"
var FOLLOWER string = "Follower"
var LEADER string = "Leader"
var TIMER_STOP string = "TIMER_STOP"
var TIMER_RESET string = "TIMER_RESET"
var HEARTBEAT_TIMEOUT int = 220
var ELECTION_TIMEOUT int =400


//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	wg sync.WaitGroup
	electionOp string
	hearbeatOp string
	majorityReceived bool
	state string
	CurrentTerm int
	VotedFor int
	Log []int
	CommitIndex int
	LastApplied int
	NextIndex []int
	MatchIndex []int

}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).

	rf.mu.Lock()
	term = rf.CurrentTerm
	if rf.state == LEADER {
		isleader = true
	} else {
		isleader = false
	}
	rf.mu.Unlock()
	return term, isleader
}


//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}


type AppendEntriesArgs struct {
 	Term int
 	LeaderId int
 	PrevLogIndex int
 	PrevLogTerm int
 	Entries[] int
 	LeaderCommit int

}

type AppendEntriesReply struct {
	Term int
	Success bool
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term int
	CandidateId int
	LastLogIndex int
	LastLogTerm int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	candidateTerm := args.Term
	myTerm := rf.CurrentTerm

	if candidateTerm < myTerm {
		reply.Term = myTerm
		reply.VoteGranted = false
		rf.log(rf.me,rf.state,rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} FAIL Term Mismatch Candidate Term at "+strconv.Itoa(args.CandidateId), false)
		return
	} else if candidateTerm > myTerm {
		// Candidate has a newer term than you
		rf.VotedFor = -1
		rf.CurrentTerm = candidateTerm
		rf.becomeFollower(" RequestVote RPC handler - Candidate has a newer term than me!")
	}

	// If Candidate term is >= mine
	if rf.VotedFor == -1 || rf.VotedFor == args.CandidateId{

		// Grant vote to Candidate
		rf.VotedFor = args.CandidateId

		reply.Term = candidateTerm
		reply.VoteGranted = true

		// Vote granted, reset my election timer

		rf.log(rf.me,rf.state,rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} PASS Candidate Term at "+strconv.Itoa(args.CandidateId), false)
		rf.log(rf.me,rf.state,rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} SIGNAL RESET election timer As VOTE GRANTED ", false)
		rf.electionOp = "RESET"
	} else if rf.VotedFor != -1 {
		rf.log(rf.me,rf.state,rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} Already Voted in this term to " + strconv.Itoa(rf.VotedFor), false)
	}



}

// Append Entry Handler
func (rf *Raft) AppendEntry(args *AppendEntriesArgs, reply *AppendEntriesReply) {

	rf.mu.Lock()
	defer rf.mu.Unlock()


	leaderTerm := args.Term
	myTerm := rf.CurrentTerm
	reply.Term = myTerm


	if leaderTerm < myTerm {
		reply.Success = false
		rf.log(rf.me,rf.state,rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} FAIL Term Mismatch Leader Term at "+strconv.Itoa(args.Term), false)
		return
	}

	rf.log(rf.me,rf.state,rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} PASS Leader Term at "+strconv.Itoa(args.Term), false)
	reply.Success = true
	rf.log(rf.me,rf.state,rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} SIGNAL RESET election timeout", false)
	rf.electionOp = "RESET"

}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntry(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntry", args, reply)
	return ok
}



//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).


	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	rf.electionOp = ""
	rf.hearbeatOp = ""
	rf.majorityReceived = false

	rf.VotedFor = -1
	rf.CurrentTerm = -1
	rf.becomeFollower("Initialize")

	return rf
}


func (rf *Raft) electionTimer() {
	var i int
	for {
		rf.mu.Lock()
		electionTimeout := rf.getRandomTimeout()
		rf.log(rf.me,rf.state,rf.CurrentTerm, " New Election Timeout = "+ strconv.Itoa(electionTimeout), false)
		rf.mu.Unlock()

		// Run election timer
		for i=1;i<=electionTimeout;i++ {
			rf.mu.Lock()
			if rf.electionOp == "STOP" {
				rf.log(rf.me,rf.state,rf.CurrentTerm, " STOPPED Election Timer at "+strconv.Itoa(i)+" ms, exiting..", false)
				rf.electionOp = ""
				rf.mu.Unlock()
				return
			} else if rf.electionOp == "RESET" {
				i = 1
				electionTimeout = rf.getRandomTimeout()
				rf.log(rf.me,rf.state,rf.CurrentTerm, " RESETTED Election Timer to "+strconv.Itoa(electionTimeout)+" ms at "+strconv.Itoa(i)+" ms", true)
				rf.electionOp = ""
				rf.mu.Unlock()
				continue
			} else {
				rf.mu.Unlock()
			}

			time.Sleep(time.Millisecond)
			rf.mu.Lock()
			if i == electionTimeout {
				rf.log(rf.me,rf.state,rf.CurrentTerm, " REACHED Election Timeout at "+strconv.Itoa(electionTimeout)+" ms", false)
				rf.mu.Unlock()
				go rf.startElection()
			} else {
				rf.mu.Unlock()
			}
		}
	}
}

func (rf *Raft) startElection()  {
	rf.mu.Lock()
	rf.log(rf.me,rf.state,rf.CurrentTerm, " START New Election", true)

	rf.becomeCandidate()

	rf.CurrentTerm += 1
	currentTerm := rf.CurrentTerm
	me := rf.me
	rf.VotedFor = me

	voteCount := 1
	majority := int(len(rf.peers)/2) + 1
	rf.mu.Unlock()

	// Send Request Vote RPC to all peers in parallel
	for  peer, _ := range rf.peers{
		if peer != me {
			go func(peer int, me int, currentTerm int) {
				rf.mu.Lock()
				requestVoteArgs := RequestVoteArgs{
					Term: currentTerm,
					CandidateId: me,
					LastLogIndex: -1,
					LastLogTerm: -1, // Need to change
				}
				requestVoteReply := RequestVoteReply{}
				rf.log(rf.me,rf.state,rf.CurrentTerm, " ASKING vote from " + strconv.Itoa(peer), false)
				rf.mu.Unlock()

				rf.sendRequestVote(peer,&requestVoteArgs, &requestVoteReply)

				// Handle the reply
				rf.mu.Lock()
				//todo should it be rf.state or state?
				rf.log(me,rf.state,currentTerm, " RECEIVED vote from " + strconv.Itoa(peer) + " Term = " + strconv.Itoa(requestVoteReply.Term) + " Result = " + strconv.FormatBool(requestVoteReply.VoteGranted), false)

				resultTerm := requestVoteReply.Term
				resultGrant := requestVoteReply.VoteGranted
				rf.mu.Unlock()

				if resultTerm > currentTerm {
					rf.mu.Lock()
					rf.CurrentTerm = resultTerm
					rf.mu.Unlock()
					rf.becomeFollower("Candidate Term is STALE")
				} else {
					if resultGrant {
						rf.mu.Lock()
						voteCount++
						rf.mu.Unlock()
					}
				}

				rf.mu.Lock()
				// ---- Vote Count ----
				if currentTerm == rf.CurrentTerm {
					if voteCount >= majority{
						if !rf.majorityReceived {
							rf.log(rf.me,rf.state,rf.CurrentTerm, " Vote Count Majority ACHIEVED :) in correct Term, Votes Got = " + strconv.Itoa(voteCount)+", Majority = " + strconv.Itoa(majority), true)
							rf.becomeLeader()
						} else {
							if requestVoteReply.VoteGranted {
								rf.log(rf.me, rf.state, rf.CurrentTerm, " Already a LEADER!", true)
							}
						}
					}
				}

				rf.mu.Unlock()

			}(peer, me, currentTerm)
		}
	}
}

func (rf *Raft) heartbeatTimer() {
	rf.log(rf.me,rf.state,rf.CurrentTerm, " STARTING Periodic Heartbeats ", true)
	var indx int
	for{

		// Run heartbeat timer
		for indx=1;indx<=HEARTBEAT_TIMEOUT;indx++ {
			rf.mu.Lock()
			if rf.hearbeatOp == "STOP" {
				rf.log(rf.me,rf.state,rf.CurrentTerm, " STOPPED Heartbeat", false)
				rf.log(rf.me,rf.state,rf.CurrentTerm, " SIGNAL STEP DOWN from LEADER to FOLLOWER..", false)
				rf.becomeFollower("STEP DOWN from LEADER to FOLLOWER")
				rf.hearbeatOp = ""
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()
			time.Sleep(time.Millisecond)
			if indx == HEARTBEAT_TIMEOUT {
				go rf.sendHeartBeat()
			}
		}
	}
}

func (rf *Raft) sendHeartBeat()  {
	rf.mu.Lock()

	leaderTerm := rf.CurrentTerm
	leaderId := rf.me

	rf.mu.Unlock()

	var wg sync.WaitGroup
	// Send Heartbeat to all peers in parallel
	for  peer, _ := range rf.peers {
		if peer != leaderId {
			wg.Add(1)
			rf.log(leaderId,rf.state,rf.CurrentTerm, " SENDING Heartbeat to "+strconv.Itoa(peer), false)
			go func(peer int, leaderId int, wg *sync.WaitGroup) {

				rf.mu.Lock()
				appendEntryArgs := AppendEntriesArgs{
					Term:         leaderTerm,
					LeaderId:     leaderId,
					PrevLogIndex: -1,
					PrevLogTerm:  -1,
					Entries:      nil,
					LeaderCommit: -1,
				}

				appendEntryReply := AppendEntriesReply{}
				rf.mu.Unlock()

				rf.sendAppendEntry(peer,&appendEntryArgs, &appendEntryReply)

				// Handle reply

				rf.mu.Lock()
				if leaderTerm < appendEntryReply.Term {
					rf.CurrentTerm = appendEntryReply.Term
					rf.hearbeatOp = "STOP"
				}
				rf.mu.Unlock()
				wg.Done()
			}(peer, leaderId, &wg)
		}
	}
	wg.Wait()
}

func (rf *Raft) becomeLeader()  {

	rf.state = LEADER
	rf.majorityReceived = true

	rf.log(rf.me,rf.state,rf.CurrentTerm, " -- YAYY I am the new LEADER " + strconv.Itoa(rf.me) +" :)", true)
	rf.log(rf.me,rf.state,rf.CurrentTerm, " SIGNAL STOP election timer " + strconv.Itoa(rf.me), true)
	rf.electionOp = "STOP"

	// Start the heartbeat timer
	go rf.heartbeatTimer()
}

func (rf *Raft) becomeCandidate()  {
	rf.log(rf.me,rf.state,rf.CurrentTerm, " Changed to CANDIDATE - " + strconv.Itoa(rf.me) +" :)", false)
	rf.state = CANDIDATE
	rf.majorityReceived = false
}

func (rf *Raft) becomeFollower(reason string)  {
	pastState := rf.state
	rf.state = FOLLOWER
	rf.majorityReceived = false

	rf.log(rf.me,rf.state,rf.CurrentTerm, " Changed to FOLLOWER - " + strconv.Itoa(rf.me) +" :( , Reason: " + reason, false)
	if pastState == LEADER || pastState == "" {
		rf.log(rf.me,rf.state,rf.CurrentTerm, " BEGIN election timer", false)
		go rf.electionTimer()
	}
}

func (rf *Raft) getRandomTimeout() int{
	return rand.Intn(50) + ELECTION_TIMEOUT
}

func (rf *Raft) log(me int, state string, term int, line string, newLine bool) {
	if newLine {
		//fmt.Println("\n[",state," ",me," at ",term, "] ",line)
	} else {
		//fmt.Println("[",state," ",me," at ",term, "] ",line)
	}
}