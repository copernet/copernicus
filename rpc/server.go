package rpc

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/websocket"
	//"github.com/btcsuite/btcd/blockchain"
	//"github.com/btcsuite/btcd/chaincfg"
	//"github.com/btcsuite/btcd/database"
	//"github.com/btcsuite/btcd/mining"
	//"github.com/btcsuite/btcd/mining/cpuminer"
	//"github.com/btcsuite/btcd/blockchain/indexers"
	"encoding/base64"
	"math/rand"
)

const (
	// rpcAuthTimeoutSeconds is the number of seconds a connection to the
	// RPC server is allowed to stay open without authenticating before it
	// is closed.
	rpcAuthTimeoutSeconds = 10

	// uint256Size is the number of bytes needed to represent an unsigned
	// 256-bit integer.
	uint256Size = 32

	// gbtNonceRange is two 32-bit big-endian hexadecimal integers which
	// represent the valid ranges of nonces returned by the getblocktemplate
	// RPC.
	gbtNonceRange = "00000000ffffffff"

	// gbtRegenerateSeconds is the number of seconds that must pass before
	// a new template is generated when the previous block hash has not
	// changed and there have been changes to the available transactions
	// in the memory pool.
	gbtRegenerateSeconds = 60

	// maxProtocolVersion is the max protocol version the server supports.
	maxProtocolVersion = 70002
)

// timeZeroVal is simply the zero value for a time.Time and is used to avoid
// creating multiple instances.
var timeZeroVal time.Time

// wsNotificationManager is a connection and notification manager used for
// websockets.  It allows websocket clients to register for notifications they
// are interested in.  When an event happens elsewhere in the code such as
// transactions being added to the memory pool or block connects/disconnects,
// the notification manager is provided with the relevant details needed to
// figure out which websocket clients need to be notified based on what they
// have registered for and notifies them accordingly.  It is also used to keep
// track of all connected websocket clients.
type wsNotificationManager struct {
	// server is the RPC server the notification manager is associated with.
	server *RRCServer

	// queueNotification queues a notification for handling.
	queueNotification chan interface{}

	// notificationMsgs feeds notificationHandler with notifications
	// and client (un)registeration requests from a queue as well as
	// registeration and unregisteration requests from clients.
	notificationMsgs chan interface{}

	// Access channel for current number of connected clients.
	numClients chan int

	// Shutdown handling
	wg   sync.WaitGroup
	quit chan struct{}
}

// rpcserverPeer represents a peer for use with the RPC server.
//
// The interface contract requires that all of these methods are safe for
// concurrent access.
type rpcserverPeer interface {
	// ToPeer returns the underlying peer instance.
	//ToPeer() *peer.Peer

	// IsTxRelayDisabled returns whether or not the peer has disabled
	// transaction relay.
	IsTxRelayDisabled() bool

	// BanScore returns the current integer value that represents how close
	// the peer is to being banned.
	BanScore() uint32

	// FeeFilter returns the requested current minimum fee rate for which
	// transactions should be announced.
	FeeFilter() int64
}

// rpcserverConnManager represents a connection manager for use with the RPC
// server.
//
// The interface contract requires that all of these methods are safe for
// concurrent access.
type rpcserverConnManager interface {
	// Connect adds the provided address as a new outbound peer.  The
	// permanent flag indicates whether or not to make the peer persistent
	// and reconnect if the connection is lost.  Attempting to connect to an
	// already existing peer will return an error.
	Connect(addr string, permanent bool) error

	// RemoveByID removes the peer associated with the provided id from the
	// list of persistent peers.  Attempting to remove an id that does not
	// exist will return an error.
	RemoveByID(id int32) error

	// RemoveByAddr removes the peer associated with the provided address
	// from the list of persistent peers.  Attempting to remove an address
	// that does not exist will return an error.
	RemoveByAddr(addr string) error

	// DisconnectByID disconnects the peer associated with the provided id.
	// This applies to both inbound and outbound peers.  Attempting to
	// remove an id that does not exist will return an error.
	DisconnectByID(id int32) error

	// DisconnectByAddr disconnects the peer associated with the provided
	// address.  This applies to both inbound and outbound peers.
	// Attempting to remove an address that does not exist will return an
	// error.
	DisconnectByAddr(addr string) error

	// ConnectedCount returns the number of currently connected peers.
	ConnectedCount() int32

	// NetTotals returns the sum of all bytes received and sent across the
	// network for all peers.
	NetTotals() (uint64, uint64)

	// ConnectedPeers returns an array consisting of all connected peers.
	ConnectedPeers() []rpcserverPeer

	// PersistentPeers returns an array consisting of all the persistent
	// peers.
	PersistentPeers() []rpcserverPeer

	// BroadcastMessage sends the provided message to all currently
	// connected peers.
	//BroadcastMessage(msg wire.Message)

	// AddRebroadcastInventory adds the provided inventory to the list of
	// inventories to be rebroadcast at random intervals until they show up
	// in a block.
	//AddRebroadcastInventory(iv *wire.InvVect, data interface{})

	// RelayTransactions generates and relays inventory vectors for all of
	// the passed transactions to all connected peers.
	//RelayTransactions(txns []*mempool.TxDesc)
}

// gbtWorkState houses state that is used in between multiple RPC invocations to
// getblocktemplate.
type gbtWorkState struct {
	sync.Mutex
	lastTxUpdate  time.Time
	lastGenerated time.Time
	prevHash      *chainhash.Hash
	minTimestamp  time.Time
	//template      *mining.BlockTemplate
	//notifyMap     map[chainhash.Hash]map[int64]chan struct{}
	//timeSource    blockchain.MedianTimeSource
}

// rpcserverSyncManager represents a sync manager for use with the RPC server.
//
// The interface contract requires that all of these methods are safe for
// concurrent access.
type rpcserverSyncManager interface {
	// IsCurrent returns whether or not the sync manager believes the chain
	// is current as compared to the rest of the network.
	IsCurrent() bool

	// SubmitBlock submits the provided block to the network after
	// processing it locally.
	//SubmitBlock(block *btcutil.Block, flags blockchain.BehaviorFlags) (bool, error)

	// Pause pauses the sync manager until the returned channel is closed.
	Pause() chan<- struct{}

	// SyncPeerID returns the ID of the peer that is currently the peer being
	// used to sync from or 0 if there is none.
	SyncPeerID() int32

	// LocateHeaders returns the headers of the blocks after the first known
	// block in the provided locators until the provided stop hash or the
	// current tip is reached, up to a max of wire.MaxBlockHeadersPerMsg
	// hashes.
	//LocateHeaders(locators []*chainhash.Hash, hashStop *chainhash.Hash) []wire.BlockHeader
}

// RpcserverConfig is a descriptor containing the RPC server configuration.
type RpcserverConfig struct {
	// Listeners defines a slice of listeners for which the RPC server will
	// take ownership of and accept connections.  Since the RPC server takes
	// ownership of these listeners, they will be closed when the RPC server
	// is stopped.
	Listeners []net.Listener

	// StartupTime is the unix timestamp for when the server that is hosting
	// the RPC server started.
	StartupTime int64

	// ConnMgr defines the connection manager for the RPC server to use.  It
	// provides the RPC server with a means to do things such as add,
	// remove, connect, disconnect, and query peers as well as other
	// connection-related data and tasks.
	//ConnMgr rpcserverConnManager

	// SyncMgr defines the sync manager for the RPC server to use.
	//SyncMgr rpcserverSyncManager

	// These fields allow the RPC server to interface with the local block
	// chain data and state.
	/*	TimeSource  blockchain.MedianTimeSource
		Chain       *blockchain.BlockChain
		ChainParams *chaincfg.Params
		DB          database.DB*/

	// TxMemPool defines the transaction memory pool to interact with.
	/*TxMemPool *mempool.TxPool*/

	// These fields allow the RPC server to interface with mining.
	//
	// Generator produces block templates and the CPUMiner solves them using
	// the CPU.  CPU mining is typically only useful for test purposes when
	// doing regression or simulation testing.
	/*	Generator *mining.BlkTmplGenerator
		CPUMiner  *cpuminer.CPUMiner*/

	// These fields define any optional indexes the RPC server can make use
	// of to provide additional data when queried.
	/*	TxIndex   *indexers.TxIndex
		AddrIndex *indexers.AddrIndex*/
}

// RRCServer provides a concurrent safe RPC server to a chain server.
type RRCServer struct {
	started                int32
	shutdown               int32
	cfg                    RpcserverConfig
	authsha                [sha256.Size]byte
	limitauthsha           [sha256.Size]byte
	ntfnMgr                *wsNotificationManager
	numClients             int32
	statusLines            map[int]string
	statusLock             sync.RWMutex
	wg                     sync.WaitGroup
	gbtWorkState           *gbtWorkState
	helpCacher             *helpCacher
	requestProcessShutdown chan struct{}
	quit                   chan int
}

func parseCmd(request *btcjson.Request) *parsedRPCCmd {
	var parsedCmd parsedRPCCmd
	parsedCmd.id = request.ID
	parsedCmd.method = request.Method

	cmd, err := btcjson.UnmarshalCmd(request)
	if err != nil {
		// When the error is because the method is not registered,
		// produce a method not found RPC error.
		if jerr, ok := err.(btcjson.Error); ok &&
			jerr.ErrorCode == btcjson.ErrUnregisteredMethod {

			parsedCmd.err = btcjson.ErrRPCMethodNotFound
			return &parsedCmd
		}

		// Otherwise, some type of invalid parameters is the
		// cause, so produce the equivalent RPC error.
		parsedCmd.err = btcjson.NewRPCError(
			btcjson.ErrRPCInvalidParams.Code, err.Error())
		return &parsedCmd
	}

	parsedCmd.cmd = cmd
	return &parsedCmd
}

// createMarshalledReply returns a new marshalled JSON-RPC response given the
// passed parameters.  It will automatically convert errors that are not of
// the type *btcjson.RPCError to the appropriate type as needed.
func createMarshalledReply(id, result interface{}, replyErr error) ([]byte, error) {
	var jsonErr *btcjson.RPCError
	if replyErr != nil {
		if jErr, ok := replyErr.(*btcjson.RPCError); ok {
			jsonErr = jErr
		} else {
			jsonErr = internalRPCError(replyErr.Error(), "")
		}
	}

	return btcjson.MarshalResponse(id, result, jsonErr)
}

// internalRPCError is a convenience function to convert an internal error to
// an RPC error with the appropriate code set.  It also logs the error to the
// RPC server subsystem since internal errors really should not occur.  The
// context parameter is only used in the log message and may be empty if it's
// not needed.
func internalRPCError(errStr, context string) *btcjson.RPCError {
	//logStr := errStr
	//if context != "" {
	//	logStr = context + ": " + errStr
	//}
	//rpcsLog.Error(logStr)
	return btcjson.NewRPCError(btcjson.ErrRPCInternal.Code, errStr)
}

// rpcDecodeHexError is a convenience function for returning a nicely formatted
// RPC error which indicates the provided hex string failed to decode.
func rpcDecodeHexError(gotHex string) *btcjson.RPCError {
	return btcjson.NewRPCError(btcjson.ErrRPCDecodeHexString,
		fmt.Sprintf("Argument must be hexadecimal string (not %q)",
			gotHex))
}

// rpcNoTxInfoError is a convenience function for returning a nicely formatted
// RPC error which indicates there is no information available for the provided
// transaction hash.
func rpcNoTxInfoError(txHash *chainhash.Hash) *btcjson.RPCError {
	return btcjson.NewRPCError(btcjson.ErrRPCNoTxInfo,
		fmt.Sprintf("No information available about transaction %v",
			txHash))
}

// httpStatusLine returns a response Status-Line (RFC 2616 Section 6.1)
// for the given request and response status code.  This function was lifted and
// adapted from the standard library HTTP server code since it's not exported.
func (s *RRCServer) httpStatusLine(req *http.Request, code int) string {
	// Fast path:
	key := code
	proto11 := req.ProtoAtLeast(1, 1)
	if !proto11 {
		key = -key
	}
	s.statusLock.RLock()
	line, ok := s.statusLines[key]
	s.statusLock.RUnlock()
	if ok {
		return line
	}

	// Slow path:
	proto := "HTTP/1.0"
	if proto11 {
		proto = "HTTP/1.1"
	}
	codeStr := strconv.Itoa(code)
	text := http.StatusText(code)
	if text != "" {
		line = proto + " " + codeStr + " " + text + "\r\n"
		s.statusLock.Lock()
		s.statusLines[key] = line
		s.statusLock.Unlock()
	} else {
		text = "status code " + codeStr
		line = proto + " " + codeStr + " " + text + "\r\n"
	}

	return line
}

// writeHTTPResponseHeaders writes the necessary response headers prior to
// writing an HTTP body given a request to use for protocol negotiation, headers
// to write, a status code, and a writer.
func (s *RRCServer) writeHTTPResponseHeaders(req *http.Request, headers http.Header, code int, w io.Writer) error {
	_, err := io.WriteString(w, s.httpStatusLine(req, code))
	if err != nil {
		return err
	}

	err = headers.Write(w)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, "\r\n")
	return err
}

// jsonRPCRead handles reading and responding to RPC messages.
func (s *RRCServer) jsonRPCRead(w http.ResponseWriter, r *http.Request, isAdmin bool) {
	fmt.Println("conf.AppConf.RPCUser", conf.AppConf.RPCUser)
	if atomic.LoadInt32(&s.shutdown) != 0 {
		return
	}

	// Read and close the JSON-RPC request body from the caller.
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		errCode := http.StatusBadRequest
		http.Error(w, fmt.Sprintf("%d error reading JSON message: %v",
			errCode, err), errCode)
		return
	}
	fmt.Println("~~~~~~~1", conf.AppConf.RPCUser)
	// Unfortunately, the http server doesn't provide the ability to
	// change the read deadline for the new connection and having one breaks
	// long polling.  However, not having a read deadline on the initial
	// connection would mean clients can connect and idle forever.  Thus,
	// hijack the connecton from the HTTP server, clear the read deadline,
	// and handle writing the response manually.
	hj, ok := w.(http.Hijacker)
	if !ok {
		errMsg := "webserver doesn't support hijacking"
		//rpcsLog.Warnf(errMsg)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		//rpcsLog.Warnf("Failed to hijack HTTP connection: %v", err)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+err.Error(), errCode)
		return
	}
	defer conn.Close()
	defer buf.Flush()
	conn.SetReadDeadline(timeZeroVal)
	// Attempt to parse the raw body into a JSON-RPC request.
	var responseID interface{}
	var jsonErr error
	var result interface{}
	var request btcjson.Request
	if err := json.Unmarshal(body, &request); err != nil {
		jsonErr = &btcjson.RPCError{
			Code:    btcjson.ErrRPCParse.Code,
			Message: "Failed to parse request: " + err.Error(),
		}
	}
	if jsonErr == nil {
		// The JSON-RPC 1.0 spec defines that notifications must have their "id"
		// set to null and states that notifications do not have a response.
		//
		// A JSON-RPC 2.0 notification is a request with "json-rpc":"2.0", and
		// without an "id" member. The specification states that notifications
		// must not be responded to. JSON-RPC 2.0 permits the null value as a
		// valid request id, therefore such requests are not notifications.
		//
		// Bitcoin Core serves requests with "id":null or even an absent "id",
		// and responds to such requests with "id":null in the response.
		//
		// Btcd does not respond to any request without and "id" or "id":null,
		// regardless the indicated JSON-RPC protocol version unless RPC quirks
		// are enabled. With RPC quirks enabled, such requests will be responded
		// to if the reqeust does not indicate JSON-RPC version.
		//
		// RPC quirks can be enabled by the user to avoid compatibility issues
		// with software relying on Core's behavior.

		/*	if request.ID == nil && !(conf.AppConf.RPCQuirks && request.Jsonrpc == "") {
			return
		}*/
		// The parse was at least successful enough to have an ID so
		// set it for the response.
		responseID = request.ID

		// Setup a close notifier.  Since the connection is hijacked,
		// the CloseNotifer on the ResponseWriter is not available.
		closeChan := make(chan struct{}, 1)
		go func() {
			_, err := conn.Read(make([]byte, 1))
			if err != nil {
				close(closeChan)
			}
		}()

		// Check if the user is limited and set error if method unauthorized
		if !isAdmin {
			if _, ok := rpcLimited[request.Method]; !ok {
				jsonErr = &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParams.Code,
					Message: "limited user not authorized for this method",
				}
			}
			_, ok = rpcLimited[request.Method]
		}
		if jsonErr == nil {
			// Attempt to parse the JSON-RPC request into a known concrete
			// command.
			parsedCmd := parseCmd(&request)
			if parsedCmd.err != nil {
				jsonErr = parsedCmd.err
			} else {
				result, jsonErr = s.standardCmdResult(parsedCmd, closeChan)
			}
		}
	}
	// Marshal the response.
	msg, err := createMarshalledReply(responseID, result, jsonErr)
	fmt.Println("msg ", msg)
	if err != nil {
		//rpcsLog.Errorf("Failed to marshal reply: %v", err)
		return
	}

	// Write the response.
	err = s.writeHTTPResponseHeaders(r, w.Header(), http.StatusOK, buf)
	if err != nil {
		//rpcsLog.Error(err)
		return
	}
	if _, err := buf.Write(msg); err != nil {
		//rpcsLog.Errorf("Failed to write marshalled reply: %v", err)
	}

	// Terminate with newline to maintain compatibility with Bitcoin Core.
	if err := buf.WriteByte('\n'); err != nil {
		//rpcsLog.Errorf("Failed to append terminating newline to reply: %v", err)
	}
}

// jsonAuthFail sends a message back to the client if the http auth is rejected.
func jsonAuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="btcd RPC"`)
	http.Error(w, "401 Unauthorized.", http.StatusUnauthorized)
}

// Start is used by server.go to start the rpc listener.
func (s *RRCServer) Start() {
	if atomic.AddInt32(&s.started, 1) != 1 {
		return
	}

	//rpcsLog.Trace("Starting RPC server")
	fmt.Println("Starting RPC server")
	rpcServeMux := http.NewServeMux()
	httpServer := &http.Server{
		Handler: rpcServeMux,

		// Timeout connections which don't complete the initial
		// handshake within the allowed timeframe.
		ReadTimeout: time.Second * rpcAuthTimeoutSeconds,
	}
	rpcServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

		// Limit the number of connections to max allowed.

		/*		if s.limitConnections(w, r.RemoteAddr) {
				return
			}*/
		// Keep track of the number of connected clients.
		s.incrementClients()
		defer s.decrementClients()
		_, isAdmin, err := s.checkAuth(r, true)
		if err != nil {
			jsonAuthFail(w)
			return
		}
		// Read and respond to the request.
		s.jsonRPCRead(w, r, isAdmin)
	})

	// Websocket endpoint.
	rpcServeMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		authenticated, isAdmin, err := s.checkAuth(r, false)
		if err != nil {
			jsonAuthFail(w)
			return
		}

		// Attempt to upgrade the connection to a websocket connection
		// using the default size for read/write buffers.
		ws, err := websocket.Upgrade(w, r, nil, 0, 0)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				//rpcsLog.Errorf("Unexpected websocket error: %v",
				//	err)
			}
			http.Error(w, "400 Bad Request.", http.StatusBadRequest)
			return
		}
		s.WebsocketHandler(ws, r.RemoteAddr, authenticated, isAdmin)
	})

	for _, listener := range s.cfg.Listeners {
		s.wg.Add(1)
		go func(listener net.Listener) {
			//rpcsLog.Infof("RPC server listening on %s", listener.Addr())
			httpServer.Serve(listener)
			//rpcsLog.Tracef("RPC listener done for %s", listener.Addr())
			s.wg.Done()
		}(listener)
	}

	s.ntfnMgr.Start()
}

// Stop is used by server.go to stop the rpc listener.
func (s *RRCServer) Stop() error {
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		//rpcsLog.Infof("RPC server is already in the process of shutting down")
		return nil
	}
	//rpcsLog.Warnf("RPC server shutting down")
	for _, listener := range s.cfg.Listeners {
		err := listener.Close()
		if err != nil {
			//rpcsLog.Errorf("Problem shutting down rpc: %v", err)
			return err
		}
	}
	s.ntfnMgr.Shutdown()
	s.ntfnMgr.WaitForShutdown()
	close(s.quit)
	s.wg.Wait()
	//rpcsLog.Infof("RPC server shutdown complete")
	return nil
}

// NewRPCServer returns a new instance of the RRCServer struct.
func NewRPCServer(config *RpcserverConfig) (*RRCServer, error) {
	rpc := RRCServer{
		cfg:         *config,
		statusLines: make(map[int]string),
		//gbtWorkState:           newGbtWorkState(config.TimeSource),
		helpCacher:             newHelpCacher(),
		requestProcessShutdown: make(chan struct{}),
		quit: make(chan int),
	}
	if conf.AppConf.RPCUser != "" && conf.AppConf.RPCPass != "" {
		login := conf.AppConf.RPCUser + ":" + conf.AppConf.RPCPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.authsha = sha256.Sum256([]byte(auth))
	}
	if conf.AppConf.RPCLimitUser != "" && conf.AppConf.RPCLimitPass != "" {
		login := conf.AppConf.RPCLimitUser + ":" + conf.AppConf.RPCLimitPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.limitauthsha = sha256.Sum256([]byte(auth))
	}
	//rpc.ntfnMgr = newWsNotificationManager(&rpc)
	//rpc.cfg.Chain.Subscribe(rpc.handleBlockchainNotification)

	return &rpc, nil
}

// RequestedProcessShutdown returns a channel that is sent to when an authorized
// RPC client requests the process to shutdown.  If the request can not be read
// immediately, it is dropped.
func (s *RRCServer) RequestedProcessShutdown() <-chan struct{} {
	return s.requestProcessShutdown
}

// NotifyNewTransactions notifies both websocket and getblocktemplate long
// poll clients of the passed transactions.  This function should be called
// whenever new transactions are added to the mempool.
func (s *RRCServer) NotifyNewTransactions(txns []*mempool.TxDesc) {
	//for _, txD := range txns {
	//	// Notify websocket clients about mempool transactions.
	//	s.ntfnMgr.NotifyMempoolTx(txD.Tx, true)
	//
	//	// Potentially notify any getblocktemplate long poll clients
	//	// about stale block templates due to the new transaction.
	//	s.gbtWorkState.NotifyMempoolTx(s.cfg.TxMemPool.LastUpdated())
	//}
}

// limitConnections responds with a 503 service unavailable and returns true if
// adding another client would exceed the maximum allow RPC clients.
//
// This function is safe for concurrent access.
func (s *RRCServer) limitConnections(w http.ResponseWriter, remoteAddr string) bool {
	if int(atomic.LoadInt32(&s.numClients)+1) > conf.AppConf.RPCMaxClients {
		//rpcsLog.Infof("Max RPC clients exceeded [%d] - "+
		//	"disconnecting client %s", cfg.RPCMaxClients,
		//	remoteAddr)
		http.Error(w, "503 Too busy.  Try again later.",
			http.StatusServiceUnavailable)
		return true
	}
	return false
}

// incrementClients adds one to the number of connected RPC clients.  Note
// this only applies to standard clients.  Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (s *RRCServer) incrementClients() {
	atomic.AddInt32(&s.numClients, 1)
}

// decrementClients subtracts one from the number of connected RPC clients.
// Note this only applies to standard clients.  Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (s *RRCServer) decrementClients() {
	atomic.AddInt32(&s.numClients, -1)
}

// checkAuth checks the HTTP Basic authentication supplied by a wallet
// or RPC client in the HTTP request r.  If the supplied authentication
// does not match the username and password expected, a non-nil error is
// returned.
//
// This check is time-constant.
//
// The first bool return value signifies auth success (true if successful) and
// the second bool return value specifies whether the user can change the state
// of the server (true) or whether the user is limited (false). The second is
// always false if the first is.
func (s *RRCServer) checkAuth(r *http.Request, require bool) (bool, bool, error) {
	authhdr := r.Header["Authorization"]
	if len(authhdr) <= 0 {
		if require {
			//rpcsLog.Warnf("RPC authentication failure from %s",
			//	r.RemoteAddr)
			//return false, false, errors.New("auth failure")
		}

		return false, false, nil
	}

	authsha := sha256.Sum256([]byte(authhdr[0]))

	// Check for limited auth first as in environments with limited users, those
	// are probably expected to have a higher volume of calls
	limitcmp := subtle.ConstantTimeCompare(authsha[:], s.limitauthsha[:])
	if limitcmp == 1 {
		return true, false, nil
	}

	// Check for admin-level auth
	cmp := subtle.ConstantTimeCompare(authsha[:], s.authsha[:])
	if cmp == 1 {
		return true, true, nil
	}

	// Request's auth doesn't match either user
	//rpcsLog.Warnf("RPC authentication failure from %s", r.RemoteAddr)
	return false, false, errors.New("auth failure")
}

// parsedRPCCmd represents a JSON-RPC request object that has been parsed into
// a known concrete command along with any error that might have happened while
// parsing it.
type parsedRPCCmd struct {
	id     interface{}
	method string
	cmd    interface{}
	err    *btcjson.RPCError
}

// standardCmdResult checks that a parsed command is a standard Bitcoin JSON-RPC
// command and runs the appropriate handler to reply to the command.  Any
// commands which are not recognized or not implemented will return an error
// suitable for use in replies.
func (s *RRCServer) standardCmdResult(cmd *parsedRPCCmd, closeChan <-chan struct{}) (interface{}, error) {
	handler, ok := rpcHandlers[cmd.method]
	fmt.Println("handler", cmd.method, handler)
	if ok {
		goto handled
	}
	// Todo: wallet
	//_, ok = rpcAskWallet[cmd.method]
	//if ok {
	//	handler = handleAskWallet
	//	goto handled
	//}
	//_, ok = rpcUnimplemented[cmd.method]
	//if ok {
	//	handler = handleUnimplemented
	//	goto handled
	//}
	return nil, btcjson.ErrRPCMethodNotFound
handled:

	return handler(s, cmd.cmd, closeChan)
}

func init() {
	rpcHandlers = rpcHandlersBeforeInit
	rand.Seed(time.Now().UnixNano())
}
