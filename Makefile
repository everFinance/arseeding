all:
	go mod tidy
	go build -o ./build/arseeding ./cmd