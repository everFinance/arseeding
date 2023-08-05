all:
	go mod tidy
	go build -o ./build/arseeding ./cmd

gen-graphql:
	go get github.com/Khan/genqlient
	genqlient ./argraphql/genqlient.yaml
