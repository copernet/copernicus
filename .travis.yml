language: go
addons:
  apt:
    sources:
      - ubuntu-toolchain-r-test
      - sourceline: 'ppa:masterminds/glide'
    packages:
      - g++-6
      - gcc-6
      - libsnappy-dev
      - zlib1g-dev
      - libbz2-dev
      - glide
go:
  - 1.10.x

cache:
  ccache: true
  apt: true
  directories:
    - ${TRAVIS_BUILD_DIR}/vendor
    - $HOME/.glide
    - _venv
    - $GOPATH/src
    - $GOPATH/pkg
    - depends/built
    - depends/sdk-sources
    - .autoconf

before_cache:
  - $HOME/gopath/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken $COVERALLS_TOKEN
  - find $HOME/.glide/cache -name ORIG_HEAD -exec rm {} \;
  - rm -rf $GOPATH/src/github.com/copernet/copernicus/*
  - rm -rf $GOPATH/pkg/**/github.com/copernet/copernicus

before_install:
  - export GOROOT=$(go env GOROOT)
  - export CXX="g++-6" CC="gcc-6"
  - export PATH=$PATH:$HOME/gopath/bin


install:

  - git clone https://github.com/copernet/secp256k1.git /tmp/secp256k1
  - pushd /tmp/secp256k1
  - ./autogen.sh
  - ./configure --enable-experimental --enable-module-ecdh --enable-module-recovery
  - make -j16
  - sudo make install
  - cd -

  - glide install

  - go get  github.com/alecthomas/gometalinter
  - gometalinter --install
  - go get golang.org/x/tools/cmd/cover
  - go get github.com/mattn/goveralls

script:
  - ./check.sh

notifications:
  email:
    on_success: change
    on_failure: alwayss
