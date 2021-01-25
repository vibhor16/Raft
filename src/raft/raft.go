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
	"bytes"
	"fmt"
	"labgob"
	"math/rand"
	"strconv"
	"sync"
	"time"
)
import "labrpc"

var CANDIDATE = "Candidate"
var FOLLOWER = "Follower"
var LEADER = "Leader"
var HeartbeatTimeout = 100
var ElectionTimeout =250
var enableLog = false

type LOG struct {
	Term int
	Command interface{}
}

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
	state string

	majorityReceived bool
	immediateHeartbeat bool // send one heartbeat immediately

	Log []LOG

	CurrentTerm int
	VotedFor int
	CommitIndex int
	LastApplied int
	NextIndex []int
	MatchIndex []int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	rf.mu.Lock()
	defer rf.mu.Unlock()

	var term int
	var isleader bool
	// Your code here (2A).

	term = rf.CurrentTerm
	if rf.state == LEADER {
		isleader = true
	} else {
		isleader = false
	}
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
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.CurrentTerm)
	e.Encode(rf.VotedFor)
	e.Encode(rf.Log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
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
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var CurrentTerm int
	var VotedFor int
	var Logs []LOG

	if d.Decode(&CurrentTerm) != nil{
		rf.log(rf.me,rf.state,rf.CurrentTerm, " CurrentTerm Decode Error", false)
	} else {
		rf.CurrentTerm = CurrentTerm
	}

	if d.Decode(&VotedFor) != nil{
		rf.log(rf.me,rf.state,rf.CurrentTerm, " VotedFor Decode Error", false)
	} else {
		rf.VotedFor = VotedFor
	}

	if d.Decode(&Logs) != nil{
		rf.log(rf.me,rf.state,rf.CurrentTerm, " Logs Decode Error", false)
	} else {
		rf.Log = Logs
	}

}


type AppendEntriesArgs struct {
 	Term int
 	LeaderId int
 	PrevLogIndex int
 	PrevLogTerm int
 	Entries []LOG
 	LeaderCommit int

}

type AppendEntriesReply struct {
	Term int
	Success bool
	ConflictTerm int
	ConflictIndex int
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
	reply.VoteGranted = false


	//If RPC request or response contains term T > currentTerm:
	//set currentTerm = T, convert to follower (§5.1)
	if candidateTerm > myTerm {
		// Candidate has a newer term than you
		rf.becomeFollower("{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} Candidate has a newer term than me! Term = "+strconv.Itoa(candidateTerm))
		rf.CurrentTerm = candidateTerm
	}

	reply.Term = rf.CurrentTerm
	// If Candidate term is >= mine
	if  rf.CurrentTerm == candidateTerm && (rf.VotedFor == -1 || rf.VotedFor == args.CandidateId) {

		// Check up-to-date - Prove if Candidate is more upto date than receiver
		isCandidateUptoDate := false
		meLastIndex := len(rf.Log) - 1
		meLastTerm := rf.Log[meLastIndex].Term

		rf.log(rf.me, rf.state, rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} CHECK Last Logs - Candidate Term = "+strconv.Itoa(args.LastLogTerm) + " Candidate LastIndex = "+strconv.Itoa(args.LastLogIndex) + " Me Term = " + strconv.Itoa(meLastTerm) + " Me LastIndex = " + strconv.Itoa(meLastIndex), false)

		//If the logs end with the same term, then whichever log is longer is more up-to-date.
		if args.LastLogTerm == meLastTerm {
			if args.LastLogIndex >= meLastIndex{
				isCandidateUptoDate = true
			}
		} else {
			//If the logs have last entries with different terms, then the log with the later term is more up-to-date.
			if args.LastLogTerm > meLastTerm {
				isCandidateUptoDate = true
			}
		}

		if isCandidateUptoDate {
			// Grant vote to Candidate
			rf.VotedFor = args.CandidateId

			reply.VoteGranted = true

			// Vote granted, reset my election timer
			rf.log(rf.me, rf.state, rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} PASS Candidate Term at "+strconv.Itoa(args.Term) + ", VOTE GRANTED!", false)
			rf.resetElectionTimer(-1)
		} else {
			// Not up-to-date
			reply.VoteGranted = false
			rf.log(rf.me, rf.state, rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} Candidate is not up-to-date! ", false)
		}
	}

	if rf.VotedFor != -1 {
		rf.log(rf.me,rf.state,rf.CurrentTerm, "{Vote Handler, C: "+strconv.Itoa(args.CandidateId)+"} Already Voted in this term to " + strconv.Itoa(rf.VotedFor), false)
	}
	rf.persist()

}

// Append Entry Handler
func (rf *Raft) AppendEntry(args *AppendEntriesArgs, reply *AppendEntriesReply) {

	rf.mu.Lock()
	defer rf.mu.Unlock()

	leaderTerm := args.Term
	myTerm := rf.CurrentTerm
	reply.Success = false
	reply.ConflictIndex = -2 // Default
	reply.ConflictTerm = -2

	//If RPC request or response contains term T > currentTerm:
	//set currentTerm = T, convert to follower (§5.1)
	if leaderTerm > myTerm {
		rf.becomeFollower("{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} FAIL Leader Term = "+strconv.Itoa(leaderTerm) + " > myTerm = " + strconv.Itoa(myTerm))
		rf.CurrentTerm = leaderTerm
	}

	reply.Term = rf.CurrentTerm
	if leaderTerm == myTerm {

		if rf.state != FOLLOWER {
			rf.becomeFollower("{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} Convert to follower")
		}

		// you get an AppendEntries RPC from the current leader (i.e., if the term in the AppendEntries arguments is outdated, you should not reset your timer);
		rf.resetElectionTimer(args.LeaderId)

		//  -- Backtrack --

		// PrevLogIndex out of bounds
		if args.PrevLogIndex >= len(rf.Log) {
			reply.ConflictIndex = len(rf.Log)
		} else {
			if rf.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
				reply.ConflictTerm = rf.Log[args.PrevLogIndex].Term


				for i:= args.PrevLogIndex; i>=0; i-- {
					if rf.Log[i].Term != reply.ConflictTerm {
						reply.ConflictIndex = i+1
						break
					}
				}
			}
		}

		// If there is a Conflict
		if reply.ConflictIndex != -2 {
			rf.log(rf.me, rf.state, rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} Conflict found, Conflict Index = " + strconv.Itoa(reply.ConflictIndex) + ", Conflict Term = " + strconv.Itoa(reply.ConflictTerm), true)
		}


		//  Reply false if log doesn’t contain an entry at prevLogIndex
		//  whose term matches prevLogTerm (§5.3)
		if args.PrevLogIndex == -1 || (args.PrevLogIndex < len(rf.Log) && rf.Log[args.PrevLogIndex].Term == args.PrevLogTerm) {
			//rf.log(rf.me,rf.state,rf.CurrentTerm, "INSIDE Append Log PLI = " + strconv.Itoa(args.PrevLogIndex) + " len log = " + strconv.Itoa(len(rf.Log)), false)

			// Proper append entry
			var entryIndex int

			followerLogIndex := args.PrevLogIndex + 1
			// Append Logs
			for entryIndex = 0; entryIndex < len(args.Entries); entryIndex++ {

				// Overlapping logs
				if followerLogIndex < len(rf.Log) {
					if rf.Log[followerLogIndex].Term != args.Entries[entryIndex].Term {

						// If an existing entry conflicts with a new one (same index
						// but different terms), delete the existing entry and all that
						// follow it (§5.3)
						rf.log(rf.me, rf.state, rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} Overlap at My Log Index = "+strconv.Itoa(followerLogIndex), true)
						rf.Log = rf.Log[:followerLogIndex]
						break
					}
				} else {
					break
				}
				followerLogIndex++
			}

			if entryIndex < len(args.Entries) {
				rf.Log = append(rf.Log, args.Entries[entryIndex:]...)
			}

			// --Update commitIndex
			//If leaderCommit > commitIndex, set commitIndex =
			//	min(leaderCommit, index of last new entry)
			if args.LeaderCommit > rf.CommitIndex {
				if args.LeaderCommit < (len(rf.Log) - 1) {
					rf.CommitIndex = args.LeaderCommit
				} else {
					rf.CommitIndex = len(rf.Log) - 1
				}
				rf.log(rf.me, rf.state, rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} Follower UPDATE Commit Index - "+strconv.Itoa(rf.CommitIndex), true)
			}

			reply.Success = true
			rf.log(rf.me, rf.state, rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(args.LeaderId)+"} PASS Logs appended! Leader term = "+strconv.Itoa(args.Term)+"\nMy Logs are = ", true, rf.Log)
		}
	}
	rf.persist()

}

func (rf *Raft) resetElectionTimer(leaderId int) {
	rf.log(rf.me,rf.state,rf.CurrentTerm, "{AppEntry Handler, L: "+strconv.Itoa(leaderId)+"} SIGNAL RESET election timeout", false)
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
	rf.mu.Lock()
	defer rf.mu.Unlock()
	index := -1
	term := -1
	isLeader := rf.state == LEADER

	// Your code here (2B).

	if !isLeader {
		return index, term, isLeader
	}

	rf.log(rf.me,rf.state,rf.CurrentTerm, " --- NEW_ENTRY = " + strconv.Itoa(command.(int)), false)
	// Append the new log
	rf.Log = append(rf.Log,
		LOG{
			Command: command,
			Term:    rf.CurrentTerm,
		})

	rf.persist()

	rf.log(rf.me,rf.state,rf.CurrentTerm, " Leader found! ", true)

	index = len(rf.Log) - 1
	return index, rf.CurrentTerm, true
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.hearbeatOp = "STOP"
	rf.electionOp = "STOP"
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
	rf.electionOp = ""
	rf.hearbeatOp = ""
	rf.majorityReceived = false
	rf.immediateHeartbeat = false

	rf.VotedFor = -1
	rf.CurrentTerm = -1
	rf.CommitIndex = 0
	rf.LastApplied = 0
	rf.NextIndex = make([]int, len(rf.peers))
	rf.MatchIndex = make([]int, len(rf.peers))

	// Dummy entry at index 0
	rf.Log = append(rf.Log, LOG{
		Term:    -1,
		Command: nil,
	})

	rf.mu.Lock()
	rf.becomeFollower("Initialize")
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.mu.Unlock()

	go rf.startLogApplyService(applyCh)



	return rf
}

func (rf *Raft) startLogApplyService(applych chan ApplyMsg){

	rf.mu.Lock()
	rf.log(rf.me,rf.state,rf.CurrentTerm, " START Apply Channel service", false)
	rf.mu.Unlock()

	for {
		//If commitIndex > lastApplied: increment lastApplied, apply
		//log[lastApplied] to state machine (§5.3)
		rf.mu.Lock()
		if rf.CommitIndex > rf.LastApplied {
			rf.LastApplied++
			lastApplied := rf.LastApplied
			command := rf.Log[rf.LastApplied].Command

			//rf.mu.Unlock()

			applych <- ApplyMsg{
				CommandValid: true,
				Command:      command,
				CommandIndex: lastApplied,
			}

			rf.log(rf.me,rf.state,rf.CurrentTerm, "--- SUCCESS Apply Channel service, CommitIndex = "+strconv.Itoa(rf.CommitIndex) + ", LastApplied = " + strconv.Itoa(rf.LastApplied), true)
		}
		rf.mu.Unlock()

		time.Sleep(time.Millisecond)
	}

}

func (rf *Raft) electionTimer() {
	for {
		rf.mu.Lock()
		electionTimeout := rf.getRandomTimeout()
		rf.log(rf.me,rf.state,rf.CurrentTerm, " New Election Timeout = "+ strconv.Itoa(electionTimeout), false)
		rf.mu.Unlock()

		// Run election timer
		for i:=1;i<=electionTimeout;i++ {

			//fmt.Println("\tElection me = ",rf.me," term = ",rf.CurrentTerm, " indx = ",i)
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
			}
			rf.mu.Unlock()

			time.Sleep(time.Millisecond)

			rf.mu.Lock()
			if i == electionTimeout && !rf.isLeader() {
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
	savedCurrTerm := rf.CurrentTerm

	me := rf.me
	rf.VotedFor = me

	lastLogIndex := len(rf.Log) - 1
	lastLogTerm := rf.Log[lastLogIndex].Term

	voteCount := 1
	majority := len(rf.peers)/2 + 1

	requestVoteArgs := RequestVoteArgs{
		Term: savedCurrTerm,
		CandidateId: me,
		LastLogIndex: lastLogIndex,
		LastLogTerm: lastLogTerm,
	}
	rf.mu.Unlock()

	// Send Request Vote RPC to all peers in parallel
	for  peer, _ := range rf.peers{

		if peer != me {
			go func(peer int, me int, requestVoteArgs RequestVoteArgs) {
				rf.mu.Lock()


				requestVoteReply := RequestVoteReply{}
				rf.log(rf.me,rf.state,rf.CurrentTerm, " ASKING vote from " + strconv.Itoa(peer), false)
				rf.printRequestVote(requestVoteArgs)
				rf.mu.Unlock()

				rf.sendRequestVote(peer,&requestVoteArgs, &requestVoteReply)

				// Handle the reply
				rf.mu.Lock()

				resultTerm := requestVoteReply.Term
				resultGrant := requestVoteReply.VoteGranted

				// 3. Reply Term Check
				if rf.isCandidate() {
					if resultTerm > savedCurrTerm {
						rf.becomeFollower("{OLD TERM} I am an expired CANDIDATE, new term = " + strconv.Itoa(resultTerm))
						rf.CurrentTerm = resultTerm
						rf.mu.Unlock()
						return
					} else if resultTerm == savedCurrTerm {
						rf.log(me, rf.state, savedCurrTerm, " RECEIVED vote from "+strconv.Itoa(peer)+" Term = "+strconv.Itoa(requestVoteReply.Term)+" Result = "+strconv.FormatBool(requestVoteReply.VoteGranted), false)

						if resultGrant {
							voteCount++
							// ---- Vote Count ----
							if voteCount >= majority {
								if !rf.majorityReceived {
									rf.log(rf.me, rf.state, rf.CurrentTerm, " Vote Count Majority ACHIEVED :) in correct Term, Votes Got = "+strconv.Itoa(voteCount)+", Majority = "+strconv.Itoa(majority), true)
									rf.becomeLeader()
								} else {
									if requestVoteReply.VoteGranted {
										rf.log(rf.me, rf.state, rf.CurrentTerm, " Already a LEADER!", true)
									}
								}
							}
						}
					}
				}
				rf.mu.Unlock()

			}(peer, me, requestVoteArgs)
		}
	}
}

func (rf *Raft) heartbeatTimer() {
	rf.mu.Lock()
	rf.log(rf.me,rf.state,rf.CurrentTerm, " STARTING Periodic Heartbeats ", true)
	rf.mu.Unlock()

	for{
		// Run heartbeat timer
		for indx := 1;indx <= HeartbeatTimeout; indx++ {
			rf.mu.Lock()
			if indx == 1 {
				go rf.prepareAndSendAppendEntry()
			}
			//fmt.Println("\tHeartbeat L=",rf.me," time - ",indx)
			if rf.hearbeatOp == "STOP" {
				rf.log(rf.me,rf.state,rf.CurrentTerm, " STOPPED Heartbeat", false)
				rf.hearbeatOp = ""
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()
			time.Sleep(time.Millisecond)
		}
	}
}

func (rf *Raft) prepareAndSendAppendEntry() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	savedCurrTerm := rf.CurrentTerm

	rf.log(rf.me,rf.state,rf.CurrentTerm, "Leader Logs are", false, rf.Log)

	var wg sync.WaitGroup
	for  peer, _ := range rf.peers {
		if peer != rf.me {
				prevLogIndex := rf.NextIndex[peer] - 1
				prevLogTerm := rf.Log[prevLogIndex].Term

				appendEntryArgs := AppendEntriesArgs{
					Term:         savedCurrTerm,
					LeaderId:     rf.me,
					PrevLogIndex: prevLogIndex,
					PrevLogTerm:  prevLogTerm,
					Entries:      nil,
					LeaderCommit: rf.CommitIndex,
				}

				lastLogIndex := len(rf.Log) - 1
				go func(peer int, appendEntryArgs AppendEntriesArgs, wg *sync.WaitGroup) {

					rf.mu.Lock()
					appendEntryReply := AppendEntriesReply{}

					// Proper append entry
					if lastLogIndex >= rf.NextIndex[peer] {
						appendEntryArgs.Entries = rf.Log[rf.NextIndex[peer]:]
						rf.log(appendEntryArgs.LeaderId, rf.state, rf.CurrentTerm, " SENDING APPEND_ENTRY to "+strconv.Itoa(peer)+", NextIndex = "+strconv.Itoa(rf.NextIndex[peer]), false)
					} else {
						// Just a heartbeat
						appendEntryArgs.Entries = nil
						rf.log(appendEntryArgs.LeaderId, rf.state, rf.CurrentTerm, " SENDING Heartbeat to "+strconv.Itoa(peer)+", NextIndex = "+strconv.Itoa(rf.NextIndex[peer]), false)
					}

					rf.printAppendEntries(appendEntryArgs)
					rf.mu.Unlock()

					rf.sendAppendEntry(peer, &appendEntryArgs, &appendEntryReply)

					// ----- Handle reply -----
					rf.mu.Lock()

					// 3. Reply Term Check
					if  appendEntryReply.Term > savedCurrTerm {
						//rf.log(appendEntryArgs.LeaderId, rf.state, rf.CurrentTerm, "The term has changed! This term =  " + strconv.Itoa(appendEntryReply.Term), false)
						//rf.hearbeatOp = "STOP"
						rf.becomeFollower("{OLD TERM} I am an expired leader, new term = " + strconv.Itoa(appendEntryReply.Term))
						rf.CurrentTerm = appendEntryReply.Term
						rf.mu.Unlock()
						return
					}

					if rf.isLeader() && savedCurrTerm == appendEntryReply.Term {
						if appendEntryReply.Success {
							rf.MatchIndex[peer] = appendEntryArgs.PrevLogIndex + len(appendEntryArgs.Entries)
							rf.NextIndex[peer] = rf.MatchIndex[peer] + 1
							rf.log(appendEntryArgs.LeaderId, rf.state, rf.CurrentTerm, " SUCCESS APPEND_ENTRY to "+strconv.Itoa(peer)+" Match Index = "+strconv.Itoa(rf.MatchIndex[peer])+" Next Index = "+strconv.Itoa(rf.NextIndex[peer]), false)

							// -- Calculate Majority for commitIndex

							//If there exists an N such that N > commitIndex, a majority
							//of matchIndex[i] ≥ N, and log[N].term == currentTerm:
							//set commitIndex = N (§5.3, §5.4).
							for N := rf.CommitIndex + 1; N < len(rf.Log); N++ {
								majorityCount := 1
								majority := len(rf.peers)/2 + 1

								if rf.Log[N].Term == rf.CurrentTerm {
									for i := 0; i < len(rf.peers); i++ {
										if i != rf.me && rf.MatchIndex[i] >= N {
											majorityCount++
										}
									}

									if majorityCount >= majority {
										rf.CommitIndex = N
										rf.log(rf.me, rf.state, rf.CurrentTerm, "-- MAJORITY = "+strconv.Itoa(majority)+" Commit Index - "+strconv.Itoa(rf.CommitIndex), true)
									}
								}
							}
						} else {
							rf.log(appendEntryArgs.LeaderId, rf.state, rf.CurrentTerm, " FAIL APPEND_ENTRY to "+strconv.Itoa(peer)+" Reply term = "+strconv.Itoa(appendEntryReply.Term)+" Match Index = "+strconv.Itoa(rf.MatchIndex[peer])+" Next Index = "+strconv.Itoa(rf.NextIndex[peer]), false)

							// Conflict found
							if appendEntryReply.ConflictIndex != -2 {
								if appendEntryReply.ConflictTerm != -2 {

									found := false
									for i:=0;i<len(rf.Log)-1;i++ {

										if rf.Log[i].Term == appendEntryReply.ConflictTerm &&  rf.Log[i+1].Term != appendEntryReply.ConflictTerm {
											rf.NextIndex[peer] = i+1
											found = true
											break
										}
									}

									if !found {
										rf.NextIndex[peer] = appendEntryReply.ConflictIndex
									}

								} else {
									rf.NextIndex[peer] = appendEntryReply.ConflictIndex
								}
							}

							if rf.NextIndex[peer] < 1 {
								rf.NextIndex[peer] = 1
							}
						}
					}

					rf.mu.Unlock()
				}(peer, appendEntryArgs, &wg)
		}
	}

	//wg.Wait()
	rf.log(rf.me,rf.state,rf.CurrentTerm, " FINISH Prepare Append Entry", false)
}

func (rf *Raft) becomeLeader()  {

	rf.state = LEADER
	rf.majorityReceived = true

	rf.log(rf.me,rf.state,rf.CurrentTerm, " -- YAYY I am the new LEADER " + strconv.Itoa(rf.me) +" :)", true)
	//rf.log(rf.me,rf.state,rf.CurrentTerm, " SIGNAL STOP election timer " + strconv.Itoa(rf.me), true)
	//rf.electionOp = "STOP"
	rf.immediateHeartbeat = true

	for i:=0;i<len(rf.peers);i++ {
		rf.NextIndex[i] = len(rf.Log) // leader last log index + 1
		rf.MatchIndex[i] = 0
	}

	// Start the heartbeat timer
	go rf.heartbeatTimer()
}

func (rf *Raft) isLeader() bool{
	if rf.state == LEADER {
		return true
	}
	return false
}

func (rf *Raft) isFollower() bool{
	if rf.state == FOLLOWER {
		return true
	}
	return false
}

func (rf *Raft) isCandidate() bool{
	if rf.state == CANDIDATE {
		return true
	}
	return false
}


func (rf *Raft) becomeCandidate()  {
	rf.log(rf.me,rf.state,rf.CurrentTerm, " Changed to CANDIDATE - " + strconv.Itoa(rf.me) +" :)", false)
	rf.state = CANDIDATE
	rf.majorityReceived = false
}

func (rf *Raft) becomeFollower(reason string)  {

	pastState := rf.state

	if pastState == FOLLOWER {
		return
	}

	rf.state = FOLLOWER
	rf.majorityReceived = false
	rf.VotedFor = -1

	rf.log(rf.me,rf.state,rf.CurrentTerm, " Changed to FOLLOWER - " + strconv.Itoa(rf.me) +" :( , Reason: " + reason, false)

	if pastState == LEADER {
		rf.hearbeatOp = "STOP"
	}

	if pastState == "" {
		rf.log(rf.me,rf.state,rf.CurrentTerm, " BEGIN election timer", false)
		go rf.electionTimer()
	}
}




func (rf *Raft) getRandomTimeout() int{
	return rand.Intn(50) + ElectionTimeout
}


func (rf *Raft) log(me int, state string, term int, line string, newLine bool, entries ...[]LOG) {

	if enableLog {
		if newLine {
			fmt.Println("\n[", state, " ", me, " at ", term, "] ", line)
		} else {
			fmt.Println("[", state, " ", me, " at ", term, "] ", line)
		}

		if len(entries) > 0 {
			fmt.Println(entries)
			fmt.Println("Log Length = ", len(entries[0]))
			fmt.Println("Voted For = ", rf.VotedFor)
			fmt.Println("Commit Index = ", rf.CommitIndex)
			fmt.Println("Last Applied = ", rf.LastApplied)
			fmt.Println()
		}
	}
}

func (rf *Raft) PrintServerState(){

	rf.mu.Lock()
	defer rf.mu.Unlock()
		fmt.Println("----------------------------")
		fmt.Println(rf.state)
		fmt.Println("Server - ",rf.me)
		fmt.Println("CurrTerm - ",rf.CurrentTerm)
		entries := rf.Log
		fmt.Println(entries)
		fmt.Println("Log Length = ", len(entries))
		fmt.Println("Voted For = ", rf.VotedFor)
		fmt.Println("Commit Index = ", rf.CommitIndex)
		fmt.Println("Last Applied = ", rf.LastApplied)
		fmt.Println("Next Index = ", rf.NextIndex)
		fmt.Println("Match Index = ", rf.MatchIndex)
		fmt.Println("-------------------------")
		fmt.Println()

}

func (rf *Raft) printAppendEntries(args AppendEntriesArgs){

	if enableLog {
		fmt.Println("{")
		fmt.Println("  Term:\t\t", args.Term)
		fmt.Println("  LeaderId:\t", args.LeaderId)
		fmt.Println("  PrevLogIndex:\t", args.PrevLogIndex)
		fmt.Println("  PrevLogTerm:\t", args.PrevLogTerm)
		fmt.Println("  Entries:\t", args.Entries)
		fmt.Println("  LeaderCommit:\t", args.LeaderCommit)
		fmt.Println("}")
	}
}

func (rf *Raft) printRequestVote(args RequestVoteArgs){

	if enableLog {
		fmt.Println("\n{")
		fmt.Println("  Term:\t\t", args.Term)
		fmt.Println("  CandidateId:\t", args.CandidateId)
		fmt.Println("  LastLogIndex:\t", args.LastLogIndex)
		fmt.Println("  LastLogTerm:\t", args.LastLogTerm)
		fmt.Println("}")
		fmt.Println()
	}
}