FROM golang:latest
MAINTAINER sandy <sandy@ever.finance>


WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o ./build/arseeding ./cmd

EXPOSE 8080

CMD [ "./build/arseeding" ]

