package conf

var CFG = struct {
	RPCUser              string
	RPCPass              string
	RPCLimitUser         string
	RPCLimitPass         string
	RPCListeners         []string
	RPCCert              string
	RPCKey               string
	RPCMaxClients        int
	RPCMaxWebsockets     int
	RPCMaxConcurrentReqs int
	RPCQuirks            bool
	DisableRPC           bool
	DisableTLS           bool
}{
	RPCUser:              "rpc",
	RPCPass:              "rpc",
	RPCLimitUser:         "",
	RPCLimitPass:         "",
	RPCListeners:         []string{"127.0.0.1:8334"},
	RPCCert:              "",
	RPCKey:               "",
	RPCMaxClients:        10,
	RPCMaxWebsockets:     25,
	RPCMaxConcurrentReqs: 20,
	RPCQuirks:            false,
	DisableRPC:           false,
	DisableTLS:           true,
}
