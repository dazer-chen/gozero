package zmq
                       
// #include <zmq.h>
// #include <stdlib.h>
// #include "get_errno.h"
import "C"

import "unsafe"                                                         
import "os"   
import "strconv"     



// ******** Global ZMQ Constants ********

const (
	ZmqPoll 		  = C.ZMQ_POLL
  ZmqP2P  		  = C.ZMQ_P2P
  ZmqPub  		  = C.ZMQ_PUB
  ZmqSub  		  = C.ZMQ_SUB
  ZmqReq  		  = C.ZMQ_REQ
  ZmqRep  		  = C.ZMQ_REP
  ZmqUpstream   = C.ZMQ_UPSTREAM
	ZmqDownstream = C.ZMQ_DOWNSTREAM
)



// ********* Contexts and InitArgs **********

// ZMQ Context type
type Context interface {
	Closeable

	NewSocket(socketType int) Socket
	Terminate()
}

// libzmq context wrapper
type lzmqContext uintptr

// Arguments to zmq_init
type InitArgs struct {
	AppThreads 	int
	IoThreads 	int
	Flags				int
}

// Integer value of environment variable GOMAXPROCS if > 1, 1 otherwise
func EnvGOMAXPROCS() int {
	var maxProcs, error = strconv.Atoi(os.Getenv("GOMAXPROCS"))
  if (error == nil && maxProcs > 1) {
		return maxProcs
	} 
  return 1
}
// Sensible default init args
// AppThreads = EnvGOMAXPROCS(), IoThreads = 1, Flags = ZmqPoll
func DefaultInitArgs() InitArgs {  
	return InitArgs{AppThreads: EnvGOMAXPROCS(), IoThreads: 1, Flags: ZmqPoll}
}
                     
// Setup a program-wide thread-safe zmq context object
func InitLibZmqContext(args InitArgs) Context {
	contextPtr := C.zmq_init(
		C.int(args.AppThreads), 
		C.int(args.IoThreads), 
		C.int(args.Flags))
                      
  CondCatchError(contextPtr == nil, libZmqErrnoFun)	

	lzmqContext := lzmqContext(contextPtr)
	return lzmqContext
}                                

// Calls Terminate()
func (p lzmqContext) Close() { p.Terminate() }

// Calls zmq_term on underlying context pointer
//
// Only call once
func (p lzmqContext) Terminate() {
	ch  := make(chan interface{})
	ptr := unsafe.Pointer(p)
	if (ptr != nil) {
  	// Needs to run in separate GoRoutine to safely lock the OS Thread
  	// and synchronize via channel to know when we're done
		Thunk(func () { 
		  CondCatchError(int(C.zmq_term(ptr)) == -1, libZmqErrnoFun)
	 	  ch <- nil
		}).NewOSThread()
    // Wait for completion
	  <- ch
	}
}


// ******** Sockets ********

// ZMQ Socket type
type Socket interface{
	Closeable

	Bind(address string)
	Connect(address string)
}

// libzmq socket wrapper
type lzmqSocket uintptr

// Creates a new Socket with the given socketType
//
// Sockets only must be used from a fixed OSThread. This may be achieved
// by conveniently using Thunk.NewOSThread() or by calling runtime.LockOSThread()
func (p lzmqContext) NewSocket(socketType int) Socket {
	ptr := unsafe.Pointer(C.zmq_socket(unsafe.Pointer(p), C.int(socketType)))
	CondCatchError(ptr == nil, libZmqErrnoFun)
	return lzmqSocket(ptr)
}

// Bind server socket
func (p lzmqSocket) Bind(address string) {
	ptr    := unsafe.Pointer(p)
  c_addr := C.CString(address)
	defer C.free(unsafe.Pointer(c_addr))
  CondCatchError(C.zmq_bind(ptr, c_addr) == -1, libZmqErrnoFun)
}

// Connect client socket
func (p lzmqSocket) Connect(address string) {
	ptr    := unsafe.Pointer(p)
  c_addr := C.CString(address)
	defer C.free(unsafe.Pointer(c_addr))
  CondCatchError(C.zmq_connect(ptr, c_addr) == -1, libZmqErrnoFun)
}

// Closes this socket 
//
// Expects the executing go routine to still be locked onto an OSThread.
// May be called only once 
func (p lzmqSocket) Close() {
	CondCatchError(int(C.zmq_close(unsafe.Pointer(p))) == -1, libZmqErrnoFun)
}



// ******** LibZmq Error Handling *******

// Default ErrnoFun used for libzmq syscalls
func libZmqErrnoFun(errno os.Errno) os.Error { return errno }

// {}

