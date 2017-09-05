

## glide Package Management 
[glide](https://github.com/Masterminds/glide) is Package Management of Golang

#### install glide
 1. easiest script
 ```
 curl https://glide.sh/get | sh
 ```
 2. On Mac OSX install the latest release via Homebrew
 ```
 brew install gulde
 ```
 3. On ubuntu install from PPA
 ```
    sudo add-apt-repository ppa:masterminds/glide 
    sudo apt-get update
    sudo apt-get install glide
   ```
 4. On windows
    go get github.com/Masterminds/glide

#### install go dependency
```
 glide install
```
#### add new package
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
#### update package 
```
glide up
```
or 
```
glide update
```
#### remove package
```
glide rm github.com/Masterminds/cookoo
```
or 
```
glide remove github.com/Masterminds/cookoo
```
## Hot compilation
We use [fswatch](https://github.com/codeskyblue/fswatch),`fswatch` is a developer tool that triggers commands in response to filesystem changes
#### install fswatch
```
go get -u -v github.com/codeskyblue/fswatch
```
#### config of fswatch
use `.fsw.yml` as config of `fswatch` , When Go file changes , the fswatch will run a command
```
go build && ./copernicus
```
#### run 
run `fswatch` in directory in project:
```
fswatch
```
So , You've been using hot compilation

