package msg

import (
	"sync"

	"github.com/btcboost/copernicus/utils"
)

var (
	gwaringLock                  sync.RWMutex
	gstrMiscWarning              string
	gfLargeWorkForkFound         bool
	gfLargeWorkInvalidChainFound bool
)

func SetMiscWarning(strWarings string) {
	gwaringLock.Lock()
	defer gwaringLock.Unlock()
	gstrMiscWarning = strWarings
}

func SetfLargeWorkForkFound(flag bool) {
	gwaringLock.Lock()
	defer gwaringLock.Unlock()
	gfLargeWorkForkFound = flag
}

func SetfLargeWorkInvalidChainFound(flag bool) {
	gwaringLock.Lock()
	defer gwaringLock.Unlock()
	gfLargeWorkInvalidChainFound = flag
}

func GetfLargeWorkForkFound() bool {
	gwaringLock.RLock()
	defer gwaringLock.RUnlock()
	return gfLargeWorkForkFound
}

func GetfLargeWorkInvalidChainFound() bool {
	gwaringLock.RLock()
	defer gwaringLock.RUnlock()
	return gfLargeWorkInvalidChainFound
}

const (
	CLIENT_VERSION_IS_RELEASE = true
	DEFAULT_TESTSAFEMODE      = false
)

func GetWarnings(strFor string) string {
	var (
		strStatusBar string
		strRPC       string
		strGUI       string
	)
	gwaringLock.Lock()
	defer gwaringLock.Unlock()

	if !CLIENT_VERSION_IS_RELEASE {
		strStatusBar = "This is a pre-release test build - use at your own " +
			"risk - do not use for mining or merchant applications"
		strGUI = "This is a pre-release test build - use at your own risk - " +
			"do not use for mining or merchant applications"
	}

	if utils.GetBoolArg("-testsafemode", DEFAULT_TESTSAFEMODE) {
		strGUI = "testsafemode enabled"
		strRPC = strGUI
		strStatusBar = strRPC
	}

	// Misc warnings like out of disk space and clock is wrong
	if gstrMiscWarning != "" {
		strStatusBar = gstrMiscWarning
		strGUI += "" + gstrMiscWarning
		if len(strGUI) != 0 {
			strGUI += "\n " + gstrMiscWarning
		}
	}

	if gfLargeWorkForkFound {
		strRPC = "Warning: The network does not appear to fully " +
			"agree! Some miners appear to be experiencing issues."
		strStatusBar = strRPC
		strGUI += "\n" + "Warning: The network does not appear to fully agree! Some " +
			"miners appear to be experiencing issues."
		if len(strGUI) != 0 {
			strGUI = "\n " + "Warning: The network does not appear to fully agree! Some " +
				"miners appear to be experiencing issues."
		}
	} else if gfLargeWorkInvalidChainFound {
		strRPC = "Warning: We do not appear to fully agree with " +
			"our peers! You may need to upgrade, or other " +
			"nodes may need to upgrade."
		strStatusBar = strRPC
		strGUI += "" + "Warning: We do not appear to fully agree with our peers! You " +
			"may need to upgrade, or other nodes may need to upgrade."
		if len(strGUI) != 0 {
			strGUI += "\n " + "Warning: We do not appear to fully agree with our peers! You " +
				"may need to upgrade, or other nodes may need to upgrade."
		}
	}

	if strFor == "gui" {
		return strGUI
	} else if strFor == "statusbar" {
		return strStatusBar
	} else if strFor == "rpc" {
		return strRPC
	}

	panic("GetWarnings(): invalid parameter")
	return "error"
}
