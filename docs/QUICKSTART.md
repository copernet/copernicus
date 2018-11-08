## build secp256k1
```
git clone https://github.com/copernet/secp256k1.git
cd secp256k1
./autogen.sh
./configure --enable-experimental --enable-module-ecdh --enable-module-recovery
make -j4
make install
```

## glide Package Management
[glide](https://github.com/Masterminds/glide) is a Package Manager for Golang

#### Install glide
Universal install script
 ```
 curl https://glide.sh/get | sh
 ```
OR if custom install on Mac OSX:
 ```
 brew install glide
 ```
OR if custom install on ubuntu:
```
 sudo add-apt-repository ppa:masterminds/glide
 sudo apt-get update
 sudo apt-get install glide
```

#### Install go dependency
 ```
 glide install
 ```
 
 if you have encountered glide errors, try the following commands:
``` rm -rf vendor
 glide cc; rm -rf ~/.glide; rm -rf vendor; rm glide.lock
 glide install --force --strip-vendor
```

#### gometalinter
```bash
go get -u github.com/alecthomas/gometalinter
gometalinter --install
```

## link missing config file
```
cp conf/bitcoincash.yml ~/Library/Application\ Support/Coper/bitcoincash.yml
```

## run copernicus
```
go build && ./copernicus
```
