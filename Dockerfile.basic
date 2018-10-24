FROM golang:1.10


RUN apt-get update \ 
	&& apt-get install -y autoconf automake libtool
RUN apt-get -y install libffi-dev
RUN apt-get -y install build-essential checkinstall
RUN apt-get -y install libreadline-gplv2-dev libncursesw5-dev libssl-dev \
        libsqlite3-dev tk-dev libgdbm-dev libc6-dev libbz2-dev

WORKDIR /usr/src
RUN wget https://www.python.org/ftp/python/3.7.0/Python-3.7.0.tgz
RUN tar xzf Python-3.7.0.tgz
WORKDIR /usr/src/Python-3.7.0
RUN ls
RUN ./configure --enable-optimizations
RUN make altinstall
RUN pip3.7 install pipenv
