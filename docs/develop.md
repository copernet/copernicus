

## Recommended Dev Environment Setup

Glide Package Management
[glide](https://github.com/Masterminds/glide) is a Package Manager for Golang

#### Install glide
The easiest way to install the latest release on Mac or Linux is with the following script:
 ```
curl https://glide.sh/get | sh
 ```
On Mac OS X you can also install the latest release via Homebrew:
 ```
brew install glide
 ```
On Ubuntu Precise (12.04), Trusty (14.04), Wily (15.10) or Xenial (16.04) you can install from our PPA:
 ```
 sudo add-apt-repository ppa:masterminds/glide
 sudo apt-get update
 apt-get install glide
```
On Ubuntu Zesty (17.04) the package is called `golang-glide`.

[Binary Packages](https://github.com/Masterminds/glide/releases) are available for Mac, Linux and Windows.

For a development version it is also possible to 
```
go get github.com/Masterminds/glide
```
#### Install go dependencies
```
 glide install
```
#### Add a new package
```
glide get github.com/Masterminds/cookoo
```
or
```
glide get github.com/Masterminds/cookoo#^1.3.0
```

Alternatively, you can add package information in glide.yaml and then glide install to add it
```
package: github.com/Masterminds/cookoo
version: ~1.3.0
```
#### Update a package
```
glide up
```
or 
```
glide update`
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
We use fswatch (https://github.com/codeskyblue/fswatch) for hot compiling while developing. `fswatch` is a developer tool that triggers commands in response to filesystem changes
				
#### Install fswatch
```
go get -u -v github.com/codeskyblue/fswatch
```

#### Fswatch config 
Each time files are updated the "cmd" field in `.fsw.yml` will be executed
Default "cmd" is 
```
go build && ./copernicus
```

#### Run
cd to project root
```
fswatch
```

That's it for fswatch!
