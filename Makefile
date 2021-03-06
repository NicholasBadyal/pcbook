all: gen

gen:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative proto/*.proto

test: gen
	go test -v -cover -race ./...

server: gen
	go run cmd/server/main.go -port 8080

client:
	go run cmd/client/main.go -address 0.0.0.0:8080

clean:
	rm proto/*.go
	rm -rf tmp/*.bin tmp/*.json
	rm -rf img/*