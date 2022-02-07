FROM ubuntu:20.04
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update
RUN apt-get install -y ca-certificates openssl
RUN apt-get install -y golang-go
RUN apt-get install -y git
WORKDIR /opt/katana/
COPY ./src/ ./src
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
RUN cd src
RUN go build -o /katanad
CMD ["/katanad"]
