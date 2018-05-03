package rpc

import (
	"github.com/btcsuite/websocket"
)

// wsCommandHandler describes a callback function used to handle a specific
// command.
type wsCommandHandler func(*wsClient, interface{}) (interface{}, error)

// wsHandlers maps RPC command strings to appropriate websocket handler
// functions.  This is set by init because help references wsHandlers and thus
// causes a dependency loop.
var wsHandlers map[string]wsCommandHandler
var wsHandlersBeforeInit = map[string]wsCommandHandler{
	//"loadtxfilter":              handleLoadTxFilter,
	//"help":                      handleWebsocketHelp,
	//"notifyblocks":              handleNotifyBlocks,
	//"notifynewtransactions":     handleNotifyNewTransactions,
	//"notifyreceived":            handleNotifyReceived,
	//"notifyspent":               handleNotifySpent,
	//"session":                   handleSession,
	//"stopnotifyblocks":          handleStopNotifyBlocks,
	//"stopnotifynewtransactions": handleStopNotifyNewTransactions,
	//"stopnotifyspent":           handleStopNotifySpent,
	//"stopnotifyreceived":        handleStopNotifyReceived,
	//"rescan":                    handleRescan,
	//"rescanblocks":              handleRescanBlocks,
}

// wsClient provides an abstraction for handling a websocket client.  The
// overall data flow is split into 3 main goroutines, a possible 4th goroutine
// for long-running operations (only started if request is made), and a
// websocket manager which is used to allow things such as broadcasting
// requested notifications to all connected websocket clients.   Inbound
// messages are read via the inHandler goroutine and generally dispatched to
// their own handler.  However, certain potentially long-running operations such
// as rescans, are sent to the asyncHander goroutine and are limited to one at a
// time.  There are two outbound message types - one for responding to client
// requests and another for async notifications.  Responses to client requests
// use SendMessage which employs a buffered channel thereby limiting the number
// of outstanding requests that can be made.  Notifications are sent via
// QueueNotification which implements a queue via notificationQueueHandler to
// ensure sending notifications from other subsystems can't block.  Ultimately,
// all messages are sent via the outHandler.
type wsClient struct {
	//sync.Mutex
	//
	//// server is the RPC server that is servicing the client.
	//server *rpcServer
	//
	//// conn is the underlying websocket connection.
	//conn *websocket.Conn
	//
	//// disconnected indicated whether or not the websocket client is
	//// disconnected.
	//disconnected bool
	//
	//// addr is the remote address of the client.
	//addr string
	//
	//// authenticated specifies whether a client has been authenticated
	//// and therefore is allowed to communicated over the websocket.
	//authenticated bool
	//
	//// isAdmin specifies whether a client may change the state of the server;
	//// false means its access is only to the limited set of RPC calls.
	//isAdmin bool
	//
	//// sessionID is a random ID generated for each client when connected.
	//// These IDs may be queried by a client using the session RPC.  A change
	//// to the session ID indicates that the client reconnected.
	//sessionID uint64
	//
	//// verboseTxUpdates specifies whether a client has requested verbose
	//// information about all new transactions.
	//verboseTxUpdates bool
	//
	//// addrRequests is a set of addresses the caller has requested to be
	//// notified about.  It is maintained here so all requests can be removed
	//// when a wallet disconnects.  Owned by the notification manager.
	//addrRequests map[string]struct{}
	//
	//// spentRequests is a set of unspent Outpoints a wallet has requested
	//// notifications for when they are spent by a processed transaction.
	//// Owned by the notification manager.
	//spentRequests map[wire.OutPoint]struct{}
	//
	//// filterData is the new generation transaction filter backported from
	//// github.com/decred/dcrd for the new backported `loadtxfilter` and
	//// `rescanblocks` methods.
	//filterData *wsClientFilter
	//
	//// Networking infrastructure.
	//serviceRequestSem semaphore
	//ntfnChan          chan []byte
	//sendChan          chan wsResponse
	//quit              chan struct{}
	//wg                sync.WaitGroup
}

// AddClient adds the passed websocket client to the notification manager.
func (m *wsNotificationManager) AddClient(wsc *wsClient) {
	//m.queueNotification <- (*notificationRegisterClient)(wsc)
}

// RemoveClient removes the passed websocket client and all notifications
// registered for it.
func (m *wsNotificationManager) RemoveClient(wsc *wsClient) {
	//select {
	//case m.queueNotification <- (*notificationUnregisterClient)(wsc):
	//case <-m.quit:
	//}
}

// Start starts the goroutines required for the manager to queue and process
// websocket client notifications.
func (m *wsNotificationManager) Start() {
	//m.wg.Add(2)
	//go m.queueHandler()
	//go m.notificationHandler()
}

// WaitForShutdown blocks until all notification manager goroutines have
// finished.
func (m *wsNotificationManager) WaitForShutdown() {
	m.wg.Wait()
}

// Shutdown shuts down the manager, stopping the notification queue and
// notification handler goroutines.
func (m *wsNotificationManager) Shutdown() {
	close(m.quit)
}

// newWebsocketClient returns a new websocket client given the notification
// manager, websocket connection, remote address, and whether or not the client
// has already been authenticated (via HTTP Basic access authentication).  The
// returned client is ready to start.  Once started, the client will process
// incoming and outgoing messages in separate goroutines complete with queuing
// and asynchrous handling for long-running operations.
func newWebsocketClient(server *RRCServer, conn *websocket.Conn,
	remoteAddr string, authenticated bool, isAdmin bool) (*wsClient, error) {

	//sessionID, err := wire.RandomUint64()
	//if err != nil {
	//	return nil, err
	//}

	client := &wsClient{
		//conn:              conn,
		//addr:              remoteAddr,
		//authenticated:     authenticated,
		//isAdmin:           isAdmin,
		//sessionID:         sessionID,
		//server:            server,
		//addrRequests:      make(map[string]struct{}),
		//spentRequests:     make(map[wire.OutPoint]struct{}),
		//serviceRequestSem: makeSemaphore(cfg.RPCMaxConcurrentReqs),
		//ntfnChan:          make(chan []byte, 1), // nonblocking sync
		//sendChan:          make(chan wsResponse, websocketSendBufferSize),
		//quit:              make(chan struct{}),
	}
	return client, nil
}

// WebsocketHandler handles a new websocket client by creating a new wsClient,
// starting it, and blocking until the connection closes.  Since it blocks, it
// must be run in a separate goroutine.  It should be invoked from the websocket
// server handler which runs each new connection in a new goroutine thereby
// satisfying the requirement.
func (s *RRCServer) WebsocketHandler(conn *websocket.Conn, remoteAddr string,
	authenticated bool, isAdmin bool) {

	// Clear the read deadline that was set before the websocket hijacked
	// the connection.
	conn.SetReadDeadline(timeZeroVal)

	// Limit max number of websocket clients.
	//rpcsLog.Infof("New websocket client %s", remoteAddr)
	//if s.ntfnMgr.NumClients()+1 > cfg.RPCMaxWebsockets {
	//	rpcsLog.Infof("Max websocket clients exceeded [%d] - "+
	//		"disconnecting client %s", cfg.RPCMaxWebsockets,
	//		remoteAddr)
	//	conn.Close()
	//	return
	//}

	// Create a new websocket client to handle the new websocket connection
	// and wait for it to shutdown.  Once it has shutdown (and hence
	// disconnected), remove it and any notifications it registered for.
	client, err := newWebsocketClient(s, conn, remoteAddr, authenticated, isAdmin)
	if err != nil {
		//rpcsLog.Errorf("Failed to serve client %s: %v", remoteAddr, err)
		conn.Close()
		return
	}
	s.ntfnMgr.AddClient(client)
	//client.Start()
	//client.WaitForShutdown()
	//s.ntfnMgr.RemoveClient(client)
	//rpcsLog.Infof("Disconnected websocket client %s", remoteAddr)
}
