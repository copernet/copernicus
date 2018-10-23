FROM golang:1.10


RUN apt-get update \ 
	&& apt-get install -y autoconf automake libtool
RUN apt-get -y install libffi-dev
RUN apt-get -y install build-essential checkinstall
RUN apt-get -y install libreadline-gplv2-dev libncursesw5-dev libssl-dev \
        libsqlite3-dev tk-dev libgdbm-dev libc6-dev libbz2-dev

RUN cd /usr/src
RUN wget https://www.python.org/ftp/python/3.7.0/Python-3.7.0.tgz
RUN tar xzf Python-3.7.0.tgz
RUN cd ./Python-3.7.0
RUN ls
RUN ./configure --enable-optimizations
RUN make altinstall

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
RUN glide install
RUN go get -u github.com/alecthomas/gometalinter
RUN gometalinter --install
RUN go build
RUN go install

WORKDIR /
RUN git clone https://github.com/copernet/walle.git
RUN cp $GOPATH/bin/copernicus /walle
RUN cd /walle
RUN mkdir .venv
RUN pipenv --python 3.7
RUN pipenv install



