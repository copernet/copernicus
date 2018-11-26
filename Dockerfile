FROM copernet/copernicus:basic

WORKDIR /
RUN git clone https://github.com/copernet/secp256k1.git
WORKDIR /secp256k1
RUN ./autogen.sh
RUN ./configure --enable-experimental --enable-module-ecdh --enable-module-recovery
RUN make -j16
RUN make install
RUN cp /usr/local/lib/libsecp256k1.so.0 /usr/lib/

RUN curl https://glide.sh/get | sh
RUN go get golang.org/x/tools/cmd/cover
RUN go get github.com/mattn/goveralls

WORKDIR /go/src/github.com/copernet/
RUN git clone https://github.com/copernet/copernicus.git
WORKDIR /go/src/github.com/copernet/copernicus
RUN git checkout magnetic
RUN glide install
RUN go get -u github.com/alecthomas/gometalinter
RUN gometalinter --install
RUN go build
RUN go install

WORKDIR /
RUN git clone https://github.com/copernet/walle.git
RUN cp $GOPATH/bin/copernicus /walle
WORKDIR /walle
RUN git checkout magnetic
RUN mkdir .venv
RUN pipenv --python 3.7
RUN pipenv install

WORKDIR /



