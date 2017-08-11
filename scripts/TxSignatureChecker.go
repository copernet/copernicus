package scripts

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

func VerfySinature(vchSig []byte, pubkey *core.PublicKey, sigHash *utils.Hash) (bool, error) {
	sign, err := core.ParseDERSignature(vchSig)
	if err != nil {
		return false, err
	}
	result := sign.Verify(sigHash.GetCloneBytes(), pubkey)
	return result, nil
}

func CheckSig(signHash *utils.Hash, vchSigIn []byte, vchPubKey []byte) (bool, error) {
	if len(vchPubKey) == 0 {
		return false, errors.New("public key is nil")
	}
	if len(vchSigIn) == 0 {
		return false, errors.New("signature is nil")
	}
	publicKey, err := core.ParsePubKey(vchPubKey)
	if err != nil {
		return false, err
	}
	ret, err := VerfySinature(vchSigIn, publicKey, signHash)
	if err != nil {
		return false, err
	}
	if !ret {
		return false, errors.New("VerfySinature is failed")
	}
	return true, nil

}

func GetHashType(vchSig []byte) byte {
	if len(vchSig) == 0 {
		return 0
	}
	return vchSig[len(vchSig)-1]
}

//
//func CheckLockTime(lockTime *CScriptNum)bool  {
//
//
//}
