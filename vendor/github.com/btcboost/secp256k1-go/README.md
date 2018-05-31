# secp256k1-go

This package provides bindings (using cgo) to the upstream [https://github.com/bitcoin-core/secp256k1](libsecp256k1) C library.

It exposes several high level functions for elliptic curve operations over the 
secp256k1 curve, namely ECDSA, point & scalar operations, ECDH, and recoverable
signatures. 

## Warning

It should be mentioned that the upstream library is still experimental
and has yet to be formally released. As such, you should think twice
before installing this package. 

The currently targeted version of libsecp256k1 is the latest master commit. 

Currently two experimental libraries are also included and supported: ECDH and 
signature recovery. These are included with the default installation, and
may eventually be discontinued by the same (as has happened with Schnorr). 

## Contributing

To start developing, clone the package from github, and from the
source directory, run the following to install the package.

    git submodule update --init
    make install
    
Tests can be run by calling `make test`
Coverage can be build by calling `make coverage`
To display a HTML code coverage report, call `make coverage-html`

Please make sure to include tests for new features.  

## Rationale behind API

There have been some slight changes to the API exposed by libsecp256k1. 
This section will document conventions adopted in the design. 

#### Always return error code from libsecp256k1
There are some functions which return more than one error code, indicating
the specific failure which occurred. With this in mind, the raw error
code is always returned as the first return value. 

To help provide some meaning to the error codes, the last parameter will
be used to return reasonable error messages.

#### Use write-by-reference where upstream uses it
In functions like EcPrivkeyTweakAdd, libsecp256k1 will take a pointer
to the private key, tweaking the value in place (overwriting the original value)

To avoid making copies of secrets in memory, we allow upstream to
overwrite the original values. If the to-be-written value is a new object,
it is returned with the other return values (example: EcdsaSign)
  