## Recommended Dev Environment Setup

##glide Package Management
[glide](https://github.com/Masterminds/glide) is Package Management of Golang

#### install glide
1. Universal one liner install script
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
OR if custom install on windows
```
   	Download binary from [here](https://github.com/Masterminds/glide/releases)

#### install go dependency

```
 	glide install
```
#### add a new package
```
	glide get github.com/Masterminds/cookoo
```
	or
```
	glide get github.com/Masterminds/cookoo#^1.3.0
```
	or add package in `glide.yaml`
```
- package: github.com/Masterminds/cookoo
  version: ~1.3.0
```
#### update a package
```
	glide up
```
or
```
	glide update
```
#### remove a package
```
	glide rm github.com/Masterminds/cookoo
```
or
```
	glide remove github.com/Masterminds/cookoo
```
## Hot compilation
	We use [fswatch](https://github.com/codeskyblue/fswatch) for hot compiling while developing. `fswatch` is a developer tool that triggers commands in response to filesystem changes
#### install fswatch
```
	go get -u -v github.com/codeskyblue/fswatch
```
#### fswatch config 
	Each time file changes the "cmd" value in `.fsw.yml` will be executed.  
```
	Default "cmd" is `go build && ./copernicus`
```
#### run
	run `fswatch` in directory in project:
```
	fswatch
```

That's it for fswatch!
