
## bulid c-secp256k1
```
git clone https://github.com/bitcoin-core/secp256k1.git
cd secp256k1
./autogen.sh
./configure --enable-experimental --enable-module-ecdh --enable-module-recovery
make -j4
make install
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
 
 # if encountered some errors from glide, try the following commands:
 rm -rf vendor
 glide cc; rm -rf ~/.glide; rm -rf vendor; rm glide.lock
 glide install --force --strip-vendor
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
