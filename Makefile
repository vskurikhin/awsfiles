all: build

build:
	GOOS=linux GOARCH=amd64 go build -o ./awsfiles ./main.go