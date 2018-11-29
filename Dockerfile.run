FROM copernet/copernicus:basic

WORKDIR /
RUN git clone https://github.com/copernet/secp256k1.git
WORKDIR /secp256k1
RUN ./autoinstall.sh
RUN echo "/usr/local/lib" > /etc/ld.so.conf.d/secp256k1.conf && ldconfig

RUN curl https://glide.sh/get | sh

WORKDIR /go/src/github.com/copernet/
COPY . ./copernicus
WORKDIR /go/src/github.com/copernet/copernicus
RUN glide install
RUN go build
RUN go install

WORKDIR /go/src/github.com/copernet/copernicus/cmd/coperctl
RUN go build .
RUN go install

WORKDIR /

ENTRYPOINT ["copernicus"]