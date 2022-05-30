FROM ubuntu:20.04

ARG package_args='--allow-downgrades --allow-remove-essential --allow-change-held-packages --no-install-recommends'
RUN echo "debconf debconf/frontend select noninteractive" | debconf-set-selections && \
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get -y $package_args update && \
  apt-get -y $package_args dist-upgrade && \
  apt-get -y $package_args install curl ca-certificates gnupg tzdata git
RUN curl --location --output go.tar.gz "https://go.dev/dl/go1.16.15.linux-amd64.tar.gz" && \
  echo "77c782a633186d78c384f972fb113a43c24be0234c42fef22c2d8c4c4c8e7475  go.tar.gz" | sha256sum -c  && \
  tar -C /usr/local -xzf go.tar.gz && \
  rm go.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /go/src/github.com/swisscom/backman
COPY . .
RUN go build -o backman

FROM ubuntu:20.04
LABEL maintainer="JamesClonk <jamesclonk@jamesclonk.ch>"

ARG package_args='--allow-downgrades --allow-remove-essential --allow-change-held-packages --no-install-recommends'
RUN echo "debconf debconf/frontend select noninteractive" | debconf-set-selections && \
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get -y $package_args update && \
  apt-get -y $package_args dist-upgrade && \
  apt-get -y $package_args install curl wget ca-certificates gnupg tzdata

RUN curl https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
  echo "deb http://apt.postgresql.org/pub/repos/apt/ focal-pgdg main" > /etc/apt/sources.list.d/pgdg.list
RUN wget -qO - https://www.mongodb.org/static/pgp/server-5.0.asc | apt-key add - && \
  echo "deb [ arch=amd64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/5.0 multiverse" > /etc/apt/sources.list.d/mongodb-org-5.0.list
RUN curl -sL https://deb.nodesource.com/setup_lts.x | bash -
RUN apt-get -y $package_args update && \
  apt-get -y $package_args install mysql-client postgresql-client-12 mongodb-database-tools=100.5.2 mongodb-org-tools=5.0.6 mongodb-org-shell=5.0.6 redis-tools nodejs openssh-server bash vim && \
  apt-get clean && \
  find /usr/share/doc/*/* ! -name copyright | xargs rm -rf && \
  rm -rf \
  /usr/share/man/* /usr/share/info/* \
  /var/lib/apt/lists/* /tmp/*
RUN npm install -g npm elasticdump

RUN mongorestore --version

RUN useradd -u 2000 -mU -s /bin/bash vcap && \
  mkdir /home/vcap/app && \
  chown vcap:vcap /home/vcap/app

WORKDIR /home/vcap/app
#COPY backman ./
COPY public ./public/
COPY static ./static/
COPY --from=0 /go/src/github.com/swisscom/backman/backman ./backman

RUN chmod +x /home/vcap/app/backman
RUN chown -R vcap:vcap /home/vcap/app
USER vcap

EXPOSE 8080

CMD ["./backman"]
