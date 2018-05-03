
## bulid c-secp256k1
```
git clone https://github.com/bitcoin-core/secp256k1.git
cd secp256k1 
./autogen.sh
./configure --enable-experimental --enable-module-ecdh --enable-module-recovery 
make -j4 
make install
```

## build rocksdb v5.5+

build and install rocksdb, according doc: (https://github.com/facebook/rocksdb/blob/master/INSTALL.md), then
install  gorocksdb.

```
cd vendor/github.com/tecbot/gorocksdb/
CGO_CFLAGS="-I/usr/local/include/rocksdb" \
CGO_LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" \
go install
```

## glide Package Management 
[glide](https://github.com/Masterminds/glide) is Package Management of Golang

#### install glide
1. easiest script
 ```
 curl https://glide.sh/get | sh
 ```
2. On Mac OSX install the latest release via Homebrew
 ```
 brew install glide
 ```
 3. On ubuntu install from PPA
```
 sudo add-apt-repository ppa:masterminds/glide 
 sudo apt-get update
 sudo apt-get install glide
```
   
#### install go dependency
```
 glide install
```

#### gometalinter
```bash
go get -u github.com/alecthomas/gometalinter
gometalinter --install
```

## run copernicus

```
go build && ./copernicus
```
