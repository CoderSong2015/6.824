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
	currentState int // 0:leader 1:Follower 2:candidate
    currentTerm int
	//isLeader bool    //自己是不是leader
	votedFor int  //当前term已经投票的候选人，在新的term应该要先置空，收不到leader心跳，会变为选举时间，此时voteFor会变为0
	peerCount int //总共有多少结点
    log[]  raftlog //暂时不用
    commitLogIndex int //已经commit的日志的index
    applyLogIndex int //已经应用的日志的index

}

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
    if rf.currentState == 0{
    	isleader = true
	}else{
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
	term int //当前的term
	candidateID int //当前自己的id？
	lastLogIndex int//候选人最后的日志index
	lastLogTerm int //候选人最后一条日志的term
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	//收到投票的回复
	term int //当前的term，给候选人更新自己的term（如果候选人比follower的term还小的话）
	voteGranted bool // true同意投票，false不同意投票
}

//
// example RequestVote RPC handler.
//这个应该是被调用的rpc？所以要吧回复写在RequestVoteReply。。。好吧
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
    //是否投票？
    //1.term是否比自己的大或者相等
    //2.日志比我新我才投票
    if rf.votedFor != 0{
        //已经投过票
		reply.term = rf.currentTerm
		reply.voteGranted = false
		return
	}
    if args.term < rf.currentTerm{
    	reply.term = rf.currentTerm
    	reply.voteGranted = false
    	return
	} else if args.lastLogIndex < rf.commitLogIndex{
		reply.term = rf.currentTerm
		reply.voteGranted = false
		return
	}

    //正式投票
	reply.term = rf.currentTerm
	reply.voteGranted = true
	rf.votedFor = args.candidateID
	return
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

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesRPC, reply *AppendEntriesReply) bool {
	//调用别人的函数，rpc来获得投票
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

//follower被leader调用此函数传送心跳
func AppendEntries ( args *AppendEntriesRPC, reply *AppendEntriesReply){
    //刷新计时器
}

type AppendEntriesReply struct {
	term int   //当前的任期号
	success bool //跟随者包含了匹配上的prevLogIndex和prevLogTerm的日志时为真
}

type AppendEntriesRPC struct{
	term int
	leaderID int
	prevLogIndex int
	prevLogTerm int
	entries[] raftlog
	leaderCommit int

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
	rf.currentState = 1 // 0:leader 1:Follower 2:candidate
	rf.currentTerm = 0
	//isLeader bool    //自己是不是leader
	rf.votedFor = 0  //当前term已经投票的候选人，在新的term应该要先置空
	rf.peerCount = len(peers)  //总共有多少结点
	 //暂时不用
	rf.commitLogIndex = 0 //已经commit的日志的index
	rf.applyLogIndex = 0 //已经应用的日志的index
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
//结点先创建，当第一个term结束/或者当前term为0，就转变为candidate来发起投票

	return rf
}

func GoToElection(rf *Raft){
	//1. election clock, random value
	//设置定时器参数，若收到RPC，则销毁已经存在的定时器
	//若定时器醒来，则转为候选人开始选举

	//timeout: convert itself to candidate
}

func main_route(rf *Raft){

}

func GetRandNum() int{
	r:=rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(230)
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