FROM golang:1.10

RUN apt-get update \ 
	&& apt-get install -y autoconf automake libtool
RUN apt-get install libffi-dev
RUN curl https://glide.sh/get | sh
WORKDIR /
RUN git clone https://github.com/copernet/secp256k1.git
WORKDIR /secp256k1
RUN ./autogen.sh
RUN ./configure --enable-experimental --enable-module-ecdh --enable-module-recovery
RUN make -j4
RUN make install
WORKDIR /go/src/github.com/copernet/copernicus
COPY . .
RUN glide install
Run go get -u github.com/alecthomas/gometalinter
RUN gometalinter --install

ENTRYPOINT ["/go/src/copernet/copernicus/check.sh"]
