# COPERNICUS BUILD NOTES 

Some notes on how to build Copernicus.

## Dependencies Build Instructions

### Secp256k1

```
git clone https://github.com/copernet/secp256k1.git
cd secp256k1
./autoinstall.sh
```

### Glide Package Management

[Glide](https://github.com/Masterminds/glide) is a Package Manager for Golang

For mac OSX:
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

## Build And Run Copernicus

```
glide install
go build && ./copernicus
```

if you have encountered glide errors, try the following commands:
```
glide cc; rm -rf ~/.glide; rm -rf vendor; rm glide.lock
glide install --force --strip-vendor
```

## Docker

You can use docker to runing copernicus:

```
docker pull copernet/copernicus:tag
docker run copernet/copernicus:tag
```
If you want build with dockerfile, you can follow this step:

```
docker build -t copernet/copernicus:local -f Dockerfile.run .
docker run copernet/copernicus:local
```
### Note
When node is syncing data, docker default storage may not satisfy with it, you need to add option `-v` when you executing docker command, such as:  `docker run -v /local/disk:/root/.bitcoincash copernet/copernicus:tag`. If you use coperctl, you should add option `-p`.
