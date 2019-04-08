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
	"fmt"
	"math/rand"
	"sync"
	"time"
)
import "labrpc"

// import "bytes"
// import "labgob"



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
	me        int                 // this peer's index into peers[] 当前结点在peers中的编号

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentState int // 1:follower 2:leader 3:candidate
    currentTerm int
	//isLeader bool    //自己是不是leader
	votedFor int  //当前term已经投票的候选人，在新的term应该要先置空，收不到leader心跳，会变为选举时间，此时voteFor会变为0
	peerCount int //总共有多少结点
    log[]  raftlog //暂时不用
    commitLogIndex int //已经commit的日志的index
    applyLogIndex int //已经应用的日志的index
	electionTimerChannel chan int //used to election timeout
	rpcTimeoutChannel chan int //used to rpc timeout

}

const (
 FOLLOWER = 1
 LEADER = 2
 CANDIDATE int = 3
)


type raftlog struct {
    op int
    term int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
    term = rf.currentTerm
    if rf.currentState == LEADER{
    	isleader = true
	}else{
		isleader = false
	}
	fmt.Println("return State!",rf.me,"current term", rf.currentTerm, mapState(rf.currentState), isleader)
    return term, isleader
}

func mapState(state int)string{
	switch state {
	case 1:
		return "follower"
	case 2:
		return "leader"
	case 3:
		return "candidate"

	}
	return "wrong"
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




//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term int //当前的term
	CandidateID int //当前自己的id？
	LastLogIndex int//候选人最后的日志index
	LastLogTerm int //候选人最后一条日志的term
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	//收到投票的回复
	Term int //当前的term，给候选人更新自己的term（如果候选人比follower的term还小的话）
	VoteGranted bool // true同意投票，false不同意投票
}

//
// example RequestVote RPC handler.
//这个应该是被调用的rpc？所以要吧回复写在RequestVoteReply。。。好吧
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
    //是否投票？
    //1.term是否比自己的大或者相等
    //2.日志比我新我才投票
    fmt.Println(rf.me,"try to get lock to vote for", args.CandidateID, " state:",mapState(rf.currentState),"term:",rf.currentTerm, "----------------------------------lock")
     rf.mu.Lock()
	fmt.Println(rf.me,"start vote!")
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		rf.mu.Unlock()
		fmt.Println(rf.me,"try to get lock to vote for", args.CandidateID, " state:",mapState(rf.currentState),"term:",rf.currentTerm, "----------------------------------unlock")
		return
	}else if rf.votedFor != -1{
        //已经投过票
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		rf.mu.Unlock()
		fmt.Println(rf.me,"try to get lock to vote for", args.CandidateID, " state:",mapState(rf.currentState),"term:",rf.currentTerm, "----------------------------------unlock,has vote for", rf.votedFor)
		return
	} else if args.LastLogIndex < rf.commitLogIndex{
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		rf.mu.Unlock()
		fmt.Println(rf.me,"try to get lock to vote for", args.CandidateID, " state:",mapState(rf.currentState),"term:",rf.currentTerm, "----------------------------------unlock")
		return
	}else{
		//正式投票
		rf.votedFor = args.CandidateID
		reply.Term = rf.currentTerm
		reply.VoteGranted = true

		rf.mu.Unlock()
		fmt.Println(rf.me,"term:",rf.currentTerm,"vote for",args.CandidateID, "----------------------------------unlock")
		return
	}
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
	//调用别人的函数，rpc来获得投票
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	//调用别人的函数，rpc来获得投票
	ok := rf.peers[server].Call("Raft.AppendEntriesRPC", args, reply)
	return ok
}

//follower被leader调用此函数传送心跳
func (rf *Raft)AppendEntriesRPC ( args *AppendEntriesArgs, reply *AppendEntriesReply){
    //刷新计时器
    //如果RPC的请求或者回复的term大于自己，那么自己就转变为FOLLOWER
    fmt.Println(rf.me, "recv append entries PRC from ", args.LeaderID,"---------------------------lock")
	rf.mu.Lock()
    if args.Term < rf.currentTerm{
    	reply.Term = rf.currentTerm
    	reply.Success = false

		rf.mu.Unlock()
		fmt.Println(rf.me, "unlock recv append entries PRC from ", args.LeaderID,"---------------------------unlock")
        return
	} else{
		//只做当选处理
		fmt.Println(rf.me,"leaderID", args.LeaderID)
		reply.Term = rf.currentTerm
		reply.Success = false

		//这里可能导致死锁，lock放在下面的话可能会一直等待管道取出数据，但是另一侧管道可能也在请求锁拿不出数据，
		//原先:
		//
		//rf.rpcTimeoutChannel <- 1
		// rf.mu.Unlock
		//这里的错误是rf可能已经在follower状态，但是electionTimerChannel永远不会建立，进不了election状态（获取数据导致死锁）
        //这里要判断当前的身份，如果是候选人说明管道里已经建立，否则则发送的RPC timeout的管道
		//}
		if rf.currentState == CANDIDATE {
			rf.currentState = FOLLOWER
			rf.currentTerm = args.Term
			rf.electionTimerChannel <- 1
			rf.mu.Unlock()
			fmt.Println(rf.me,"state: CANDIDATE unlock recv append entries PRC from ", args.LeaderID,"---------------------------unlock")
			return
		}else if rf.currentState == FOLLOWER{
			//rf.currentState = FOLLOWER
			rf.currentTerm = args.Term
			rf.mu.Unlock()
			fmt.Println(rf.me,"state: FOLLOWER unlock recv append entries PRC from ", args.LeaderID,"---------------------------unlock")
			rf.rpcTimeoutChannel <- 1
			return
		}else if rf.currentState == LEADER{ //是否要term大于自己才转换？理论上应该是，但是选举确保新leader应该term大于老leader，分区?不够数量当选
			//leader应该变为follower并且要发送给leader循环
			rf.currentState = FOLLOWER
			rf.currentTerm = args.Term
			rf.rpcTimeoutChannel <- 1
			rf.mu.Unlock()
			fmt.Println(rf.me,"state: LEADER unlock recv append entries PRC from ", args.LeaderID,"---------------------------unlock")
			return
		}

	}


	fmt.Println(rf.me, "unlock recv append entries PRC from  ", args.LeaderID,"---------------------------unlock")
}

type AppendEntriesReply struct {
	Term int   //当前的任期号
	Success bool //跟随者包含了匹配上的prevLogIndex和prevLogTerm的日志时为真
}

type AppendEntriesArgs struct{
	Term int
	LeaderID int
	//PrevLogIndex int
	//PrevLogTerm int
	//Entries[] raftlog
	//LeaderCommit int
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
	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	rf.currentState = FOLLOWER // 1:follower 2: leader  2:candidate
	rf.currentTerm = 0
	//isLeader bool    //自己是不是leader
	rf.votedFor = -1  //当前term已经投票的候选人，在新的term应该要先置空
	rf.peerCount = len(peers)  //总共有多少结点
	 //暂时不用
	rf.commitLogIndex = 0 //已经commit的日志的index
	rf.applyLogIndex = 0 //已经应用的日志的index
	rf.electionTimerChannel = make(chan int)
	rf.rpcTimeoutChannel = make(chan int)

	var rl raftlog
	rl.term = 0
	rl.op = 0
	rf.log= make([]raftlog, 0, 100)
	rf.log = append(rf.log, rl)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
//结点先创建，当第一个term结束/或者当前term为0，就转变为candidate来发起投票
    go main_route(rf)
	return rf
}
//func timed( t *time.Timer, stopChannel chan int) {
//
//	//chf:= make(chan int)
//	go func() {
//		select {
//		case <-t.C:
//			fmt.Println("子协程可以打印了，因为定时器的时间到")
//		case <-stopChannel: //新增加一个关闭标志，如果是这个关闭，则直接return
//			//ch <- 1
//			return
//		}
//	}()
//
//	//等待完成
//	//<-chf
//}

func GoToElection(rf *Raft){
	//1. election clock, random value
	fmt.Println(rf.me,"try to get vote lock to be candidate, term",rf.currentTerm,"---------------------------lock")
	rf.mu.Lock()
	rf.votedFor = -1
	rf.mu.Unlock()
	fmt.Println(rf.me,"finish get vote lock to be candidate",rf.currentTerm,"---------------------------unlock")
	//chf:= make(chan int)
	//t := time.NewTimer(time.Duration(GetRandNum()) * time.Millisecond)
	select {
		case <-time.After(time.Duration(GetRandNum(rf.me) * 5) * time.Millisecond):
			//定时器时间到了，没收到回复，转为candidate
			for{
				fmt.Println(rf.me,"go into candidate Election",rf.currentTerm, time.Now())
				if candidateElection(rf){
					//收到新leader信息或者称为leader
					fmt.Println(rf.me,"my State", mapState(rf.currentState))
					break
				}
			}

		case <-rf.electionTimerChannel: //新增加一个关闭标志，如果是这个关闭，则直接return，推出协程
		    //1、收到投票信息并投出去了 2、收到新leader信息
			//chf <- 0
			fmt.Println(rf.me,rf.currentTerm,rf.votedFor)
			return
	}
}


//选举失败返回 false
//成为follower或者成为leader 返回ok
func candidateElection(rf * Raft) bool{
	fmt.Println(rf.me,"tryto get candidate lock", rf.currentTerm,"---------------------------lock")
	rf.mu.Lock()
	rf.currentTerm += 1
	rf.currentState = CANDIDATE
	rf.votedFor = rf.me
	rf.mu.Unlock()
	fmt.Println(rf.me,"ok get candidate lock", rf.currentTerm,"---------------------------unlock")
	//chf:= make(chan int)
	//启动收集选票线程
	electchan := make(chan int)
	go func(rf *Raft, electchan chan int){
		var rvags RequestVoteArgs
        var reply RequestVoteReply
		fmt.Println(rf.me,"go into election, currnet term:", rf.currentTerm,"---------------------------lock")
		rf.mu.Lock()
		rvags.Term = rf.currentTerm
		rvags.CandidateID = rf.me
		rvags.LastLogIndex = rf.commitLogIndex
		rvags.LastLogTerm = rf.log[rf.commitLogIndex].term

		rf.mu.Unlock()
		fmt.Println(rf.me,"unlock go into election, currnet term:", rf.currentTerm,"---------------------------unlock")
        reply.VoteGranted = false
        reply.Term = 0
        //算上自己的一票。。。
		voteCount:=1
		//fmt.Println("go into vote",rf.me, rf.currentTerm)
		for i:=0; i < rf.peerCount;i++{
			if i != rf.me && rf.sendRequestVote(i, &rvags, &reply){
				fmt.Println(rf.me, "recvone from",i, reply.VoteGranted,"term",reply.Term)
				if reply.VoteGranted{
					voteCount += 1
				}
			}else{
				if i!= rf.me{
					fmt.Println(rf.me, "candidate recive ",i, "term",reply.Term, "wrong-----------no response")
				}
			}

			if voteCount > rf.peerCount / 2{
				//宣布当选
				//fmt.Println("go leader1",rf.me, voteCount,rf.currentTerm)
				electchan <- 1
				//break
			}
		}
	}(rf, electchan)
	select {
		case <-time.After(time.Duration(GetRandNum(rf.me)) * time.Millisecond):
			//定时器时间到了，没收到回复或者成功当选leader
			fmt.Println(rf.me,"candidate time out")
			return false
		case <-rf.electionTimerChannel: //新增加一个关闭标志，如果是这个关闭，则直接return，推出协程
			//1、收到投票信息并投出去了 2、收到新leader信息
			//handle directly in rpc
			fmt.Println(rf.me,"has vote!exit candidate, term:", rf.currentTerm)
			return true
	    case <-electchan:
	    	//当选为leader
            var aerpc AppendEntriesArgs
			fmt.Println(rf.me,"go leader, term:",rf.currentTerm)
			fmt.Println(rf.me,"to be leader", rf.currentTerm,"---------------------------lock")
	    	rf.mu.Lock()
            rf.currentState = LEADER
            aerpc.Term = rf.currentTerm
            aerpc.LeaderID = rf.me
            rf.mu.Unlock()
			fmt.Println(rf.me,"unlock to be leader", rf.currentTerm,"---------------------------unlock")

            var reply AppendEntriesReply
			for i:=0; i < rf.peerCount;i++{
				if i != rf.me{

					rf.sendAppendEntries(i, &aerpc, &reply)
					fmt.Println(rf.me,"send to",i,"leader rpc ok, term",rf.currentTerm)
					if reply.Term > rf.currentTerm{
						fmt.Println("leader",rf.me, "term is small, goto follower","--------------------------lock")
						rf.mu.Lock()
						rf.currentState = FOLLOWER
						rf.currentTerm = reply.Term
						rf.mu.Unlock()
						fmt.Println("leader",rf.me, "term is small, goto follower","--------------------------unlock")
						return true
					}

				}
			}

	    	return true
	}


	return true
}

func main_route(rf *Raft){
    //每个term一开始就进入选举
    //每个term 10分钟？
    time.Sleep(time.Duration(GetRandNum(rf.me))*time.Millisecond)
	for {
		timer := time.NewTimer(600 * time.Second) //term time
		GoToElection(rf)
		NewWork:
		if rf.currentState==FOLLOWER {
		loops:
			for {
				fmt.Println(rf.me, "go follower! term:",rf.currentTerm)
				rpctimer := time.NewTimer(1000 * time.Millisecond)
				select {
				case <-timer.C:
					//一个term结束，开始下一个term
					break
				case <-rpctimer.C:
					//RPC timeout，continue进入选举准备
					fmt.Println(rf.me, "follower rpc timeout! term:", rf.currentTerm)
					break loops
				case <-rf.rpcTimeoutChannel:
					fmt.Println(rf.me, "get leader rpc,reset timer, term:", rf.currentTerm)
					rpctimer.Reset(0)
					continue
				}
			}

		}else if rf.currentState==LEADER{
			fmt.Println(rf.me, "leader go!term:",rf.currentTerm)
			for {
				rpctimer := time.NewTimer(130 * time.Millisecond)
				select {
				case <-timer.C:
					//一个term结束，开始下一个term
					fmt.Println(rf.me, "finish one term:",rf.currentTerm)
					break

				case <-rf.rpcTimeoutChannel:
					//以leader的身份接受到此channel说明有term更高的leader发送RPC，此时转为FOLLOWER工作
					fmt.Println(rf.me, "leader go into follower", rf.currentTerm)
					//rpctimer.Reset(0)
					goto NewWork
				case <-rpctimer.C:
					//RPC timeout，发送rpc
					var aerpc AppendEntriesArgs
					//fmt.Println("go leader333",rf.me, rf.currentTerm)
					//rf.currentState = LEADER
					aerpc.Term = rf.currentTerm
					aerpc.LeaderID = rf.me

					var reply AppendEntriesReply
					for i:=0; i < rf.peerCount;i++{
						if i != rf.me{
							fmt.Println("leader",rf.me, "send heartbreak rpc to",i," term",rf.currentTerm)
							if rf.sendAppendEntries(i, &aerpc, &reply){
								fmt.Println("leader",rf.me, "recv ",i,"'s term:", reply.Term                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               )
								if reply.Term > rf.currentTerm{
									fmt.Println("leader",rf.me, "term is small, goto follower","--------------------------lock")
									rf.mu.Lock()
									rf.currentState = FOLLOWER
									rf.currentTerm = reply.Term
									rf.mu.Unlock()
									fmt.Println("leader",rf.me, "term is small, goto follower","--------------------------unlock")
									goto NewWork
								}

							}else{
								fmt.Println("leader",rf.me, "send heartbreak rpc to",i," term",rf.currentTerm,"----no response")
							}

						}
					}
				}
			}
		}
	}



}

func GetRandNum(peerNumber int) int{
//	r:=rand.New(rand.NewSource(time.Now().UnixNano()))
//	fmt.Println("random ",300 + r.Intn(150))
	//k:= 100 +r.Intn(350)
	t1:=time.Now().UnixNano()//取到当前时间，精确到纳秒
	//时间戳：当前的时间，距离1970年1月1日0点0分0秒的秒值，纳秒值
	rand.Seed(t1)//把当前时间作为种子传给计算机

	k:= 150 + rand.Intn(100 + peerNumber*100) + peerNumber * 50
    fmt.Println("random:",k)
	return k
}
//
//fmt.Println()
//r:=rand.New(rand.NewSource(time.Now().UnixNano()))
//fmt.Print(r.Intn(230))
////var name string
////fmt.Scanf("%s", &name)
////fmt.Print(name)
//timer := time.NewTimer(10 * time.Second)
//
//stopChannel := make(chan int)
//go timed(timer,stopChannel)
//
//
//var name string
//for true {
//fmt.Scanf("%s", &name)
//
//if name == "hao" {
//close(stopChannel)
//timer.Stop()
//}
//}
//}
//
//func timed( t *time.Timer, stopChannel chan int) {
//
//	//chf:= make(chan int)
//	go func() {
//		select{
//		case <-t.C:
//			fmt.Println("子协程可以打印了，因为定时器的时间到")
//		case <-stopChannel: //新增加一个关闭标志，如果是这个关闭，则直接return
//			//ch <- 1
//			return
//		}
//	}()
//	//等待完成
//	//<-chf
//	fmt.Print("finish")
//
//}