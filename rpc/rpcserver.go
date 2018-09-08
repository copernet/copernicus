package rpc

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/rpc/btcjson"
)

const (
	// rpcAuthTimeoutSeconds is the number of seconds a connection to the
	// RPC server is allowed to stay open without authenticating before it
	// is closed.
	rpcAuthTimeoutSeconds = 10
)

func internalRPCError(errStr, context string) *btcjson.RPCError {
	logStr := errStr
	if context != "" {
		logStr = context + ": " + errStr
	}
	log.Error(logStr)
	return btcjson.NewRPCError(btcjson.ErrRPCInternal.Code, errStr)
}

/**
 * convenience function for returning a nicely formatted
 * RPC error which indicates the provided hex string failed to decode.
 */
func rpcDecodeHexError(gotHex string) *btcjson.RPCError {
	return btcjson.NewRPCError(btcjson.ErrRPCDecodeHexString,
		fmt.Sprintf("Argument must be hexadecimal string (not %q)",
			gotHex))
}

// Server provides a concurrent safe RPC server to a chain server.
type Server struct {
	started                int32
	shutdown               int32
	cfg                    ServerConfig
	authsha                [sha256.Size]byte
	limitauthsha           [sha256.Size]byte
	numClients             int32
	statusLines            map[int]string
	statusLock             sync.RWMutex
	wg                     sync.WaitGroup
	helpCacher             *helpCacher
	requestProcessShutdown chan struct{}
	quit                   chan int
}

func (s *Server) httpStatusLine(req *http.Request, code int) string {
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

func (s *Server) writeHTTPResponseHeaders(req *http.Request, headers http.Header, code int, w io.Writer) error {
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

// Stop is used by server.go to stop the rpc listener.
func (s *Server) Stop() error {
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		log.Info("RPC server is already in the process of shutting down")
		return nil
	}
	log.Warn("RPC server shutting down")
	for _, listener := range s.cfg.Listeners {
		err := listener.Close()
		if err != nil {
			log.Error("Problem shutting down rpc: %v", err)
			return err
		}
	}
	close(s.quit)
	s.wg.Wait()
	log.Info("RPC server shutdown complete")
	return nil
}

// RequestedProcessShutdown returns a channel that is sent to when an authorized
// RPC client requests the process to shutdown.  If the request can not be read
// immediately, it is dropped.
func (s *Server) RequestedProcessShutdown() <-chan struct{} {
	return s.requestProcessShutdown
}

// responds with a 503 service unavailable and returns true if
// adding another client would exceed the maximum allow RPC clients.
func (s *Server) limitConnections(w http.ResponseWriter, remoteAddr string) bool {
	if int(atomic.LoadInt32(&s.numClients)+1) > conf.Cfg.RPC.RPCMaxClients {
		log.Info("Max RPC clients exceeded [%d] - "+
			"disconnecting client %s", conf.Cfg.RPC.RPCMaxClients,
			remoteAddr)
		http.Error(w, "503 Too busy.  Try again later.",
			http.StatusServiceUnavailable)
		return true
	}
	return false
}

//  adds one to the number of connected RPC clients.
func (s *Server) incrementClients() {
	atomic.AddInt32(&s.numClients, 1)
}

//  subtracts one from the number of connected RPC clients.
func (s *Server) decrementClients() {
	atomic.AddInt32(&s.numClients, -1)
}

func (s *Server) checkAuth(r *http.Request, require bool) (bool, bool, error) {
	authhdr := r.Header["Authorization"]
	if len(authhdr) <= 0 {
		if require {
			log.Warn("RPC authentication failure from %s",
				r.RemoteAddr)
			return false, false, errors.New("auth failure")
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
	log.Warn("RPC authentication failure from %s", r.RemoteAddr)
	return false, false, errors.New("auth failure")
}

// JSON-RPC request object that has been parsed into a known concrete command
// along with any error that might have happened while parsing it.
type parsedRPCCmd struct {
	id     interface{}
	method string
	cmd    interface{}
	err    *btcjson.RPCError
}

func (s *Server) standardCmdResult(cmd *parsedRPCCmd, closeChan <-chan struct{}) (interface{}, error) {
	handler, ok := rpcHandlers[cmd.method]
	if ok {
		goto handled
	}
handled:
	return handler(s, cmd.cmd, closeChan)
}

func parseCmd(request *btcjson.Request) *parsedRPCCmd {
	var parsedCmd parsedRPCCmd
	parsedCmd.id = request.ID
	parsedCmd.method = request.Method

	cmd, err := btcjson.UnmarshalCmd(request)
	if err != nil {
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

// jsonRPCRead handles reading and responding to RPC messages.
func (s *Server) jsonRPCRead(w http.ResponseWriter, r *http.Request, isAdmin bool) {
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

	// Unfortunately, the http server doesn't provide the ability to
	// change the read deadline for the new connection and having one breaks
	// long polling.  However, not having a read deadline on the initial
	// connection would mean clients can connect and idle forever.  Thus,
	// hijack the connecton from the HTTP server, clear the read deadline,
	// and handle writing the response manually.
	hj, ok := w.(http.Hijacker)
	if !ok {
		errMsg := "webserver doesn't support hijacking"
		log.Warn(errMsg)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		log.Warn("Failed to hijack HTTP connection: %v", err)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+err.Error(), errCode)
		return
	}
	defer conn.Close()
	defer buf.Flush()
	//conn.SetReadDeadline(timeZeroVal)

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
		if request.ID == nil && !(conf.Cfg.RPC.RPCQuirks && request.Jsonrpc == "") {
			return
		}

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
		//if !isAdmin {
		//	if _, ok := rpcLimited[request.Method]; !ok {
		//		jsonErr = &btcjson.RPCError{
		//			Code:    btcjson.ErrRPCInvalidParams.Code,
		//			Message: "limited user not authorized for this method",
		//		}
		//	}
		//}

		if jsonErr == nil {
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
	if err != nil {
		log.Error("Failed to marshal reply: %v", err)
		return
	}

	// Write the response.
	err = s.writeHTTPResponseHeaders(r, w.Header(), http.StatusOK, buf)
	if err != nil {
		log.Error(err)
		return
	}
	if _, err := buf.Write(msg); err != nil {
		log.Error("Failed to write marshalled reply: %v", err)
	}

	// Terminate with newline to maintain compatibility.
	if err := buf.WriteByte('\n'); err != nil {
		log.Error("Failed to append terminating newline to reply: %v", err)
	}
}

// jsonAuthFail sends a message back to the client if the http auth is rejected.
func jsonAuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="btcd RPC"`)
	http.Error(w, "401 Unauthorized.", http.StatusUnauthorized)
}

// Start func starts the rpc listener.
func (s *Server) Start() {
	if atomic.AddInt32(&s.started, 1) != 1 {
		return
	}

	log.Info("Starting RPC server")
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
		if s.limitConnections(w, r.RemoteAddr) {
			return
		}

		// Keep track of the number of connected clients.
		s.incrementClients()
		defer s.decrementClients()
		_, isAdmin, err := s.checkAuth(r, true)
		if err != nil {
			jsonAuthFail(w)
			return
		}

		s.jsonRPCRead(w, r, isAdmin)
	})

	for _, listener := range s.cfg.Listeners {
		s.wg.Add(1)
		go func(listener net.Listener) {
			log.Info("RPC server listening on %s", listener.Addr())
			httpServer.Serve(listener)
			log.Trace("RPC listener done for %s", listener.Addr())
			s.wg.Done()
		}(listener)
	}
}

// GenCertPair generates a key/cert pair to the paths provided.
func GenCertPair(certFile, keyFile string) error {
	log.Info("Generating TLS certificates...")

	org := "copernicus autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := NewTLSCertPair(org, validUntil, nil)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	log.Info("Done generating TLS certificates")
	return nil
}

type ServerConfig struct {
	Listeners []net.Listener
	// unix timestamp for when the server that is hosting the RPC server started.
	StartupTime int64
}

// SetupRPCListeners returns a slice of listeners that are configured for use
// with the RPC server depending on the configuration settings for listen
// addresses and TLS.
func SetupRPCListeners() ([]net.Listener, error) {
	// Setup TLS if not disabled.
	listenFunc := net.Listen
	if !conf.Cfg.P2PNet.DisableTLS {
		// Generate the TLS cert and key file if both don't already exist.
		if !fileExists(conf.Cfg.RPC.RPCKey) && !fileExists(conf.Cfg.RPC.RPCCert) {
			err := GenCertPair(conf.Cfg.RPC.RPCCert, conf.Cfg.RPC.RPCKey)
			if err != nil {
				return nil, err
			}
		}

		keypair, err := tls.LoadX509KeyPair(conf.Cfg.RPC.RPCCert, conf.Cfg.RPC.RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}

		// Change the standard net.Listen function to the tls one.
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	}

	netAddrs, err := parseListeners(conf.Cfg.RPC.RPCListeners)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			log.Warn("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// parseListeners determines whether each listen address is IPv4 and IPv6 and
// returns a slice of appropriate net.Addrs to listen on with TCP. It also
// properly detects addresses which apply to "all interfaces" and adds the
// address as both IPv4 and IPv6.
func parseListeners(addrs []string) ([]net.Addr, error) {
	netAddrs := make([]net.Addr, 0, len(addrs)*2)
	for _, addr := range addrs {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			// Shouldn't happen due to already being normalized.
			return nil, err
		}

		// Empty host or host of * on plan9 is both IPv4 and IPv6.
		if host == "" || (host == "*" && runtime.GOOS == "plan9") {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
			continue
		}

		// Strip IPv6 zone id if present since net.ParseIP does not
		// handle it.
		zoneIndex := strings.LastIndex(host, "%")
		if zoneIndex > 0 {
			host = host[:zoneIndex]
		}

		// Parse the IP.
		ip := net.ParseIP(host)
		if ip == nil {
			return nil, fmt.Errorf("'%s' is not a valid IP address", host)
		}

		// To4 returns nil when the IP is not an IPv4 address, so use
		// this determine the address type.
		if ip.To4() == nil {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
		} else {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
		}
	}
	return netAddrs, nil
}

type simpleAddr struct {
	net, addr string
}

func (a simpleAddr) String() string {
	return a.addr
}

func (a simpleAddr) Network() string {
	return a.net
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = simpleAddr{}

func NewServer(config *ServerConfig) (*Server, error) {
	rpc := Server{
		cfg:         *config,
		statusLines: make(map[int]string),
		//gbtWorkState:           newGbtWorkState(config.TimeSource), // todo open
		helpCacher:             newHelpCacher(),
		requestProcessShutdown: make(chan struct{}),
		quit:                   make(chan int),
	}
	if conf.Cfg.RPC.RPCUser != "" && conf.Cfg.RPC.RPCPass != "" {
		login := conf.Cfg.RPC.RPCUser + ":" + conf.Cfg.RPC.RPCPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.authsha = sha256.Sum256([]byte(auth))
	}
	if conf.Cfg.RPC.RPCLimitUser != "" && conf.Cfg.RPC.RPCLimitPass != "" {
		login := conf.Cfg.RPC.RPCLimitUser + ":" + conf.Cfg.RPC.RPCLimitPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.limitauthsha = sha256.Sum256([]byte(auth))
	}
	//rpc.cfg.Chain.Subscribe(rpc.handleBlockchainNotification)  // todo open

	return &rpc, nil
}

func InitRPCServer() (*Server, error) {
	if !conf.Cfg.P2PNet.DisableRPC {
		// Setup listeners for the configured RPC listen addresses and
		// TLS settings.
		rpcListeners, err := SetupRPCListeners()
		if err != nil {
			return nil, err
		}
		if len(rpcListeners) == 0 {
			return nil, errors.New("RPCS: No valid listen address")
		}

		rpcServer, err := NewServer(&ServerConfig{
			Listeners: rpcListeners,
			//StartupTime: s.startupTime,
		})
		if err != nil {
			return nil, err
		}

		return rpcServer, nil
	}
	return nil, nil
}
