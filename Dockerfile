FROM alpine:latest
MAINTAINER sandy <sandy@ever.finance>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

WORKDIR /arseeding

# 外挂bolt文件夹
VOLUME ["/arseeding/data"]

COPY cmd/arseeding /arseeding/arseeding
EXPOSE 8080

ENTRYPOINT [ "/arseeding/arseeding" ]