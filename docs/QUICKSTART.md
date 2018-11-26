## build and install secp256k1 lib
```
git clone https://github.com/copernet/secp256k1.git
cd secp256k1
./autoinstall.sh
```

## glide Package Management
[glide](https://github.com/Masterminds/glide) is a Package Manager for Golang

#### Install glide
For Mac OSX:
```
brew install glide
```

For Ubuntu:
```
sudo add-apt-repository ppa:masterminds/glide
sudo apt-get update
sudo apt-get install glide
```

For Centos:
```
sudo yum install glide
```

Universal install script
```
curl https://glide.sh/get | sh
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

## run copernicus
```
go build && ./copernicus
```
