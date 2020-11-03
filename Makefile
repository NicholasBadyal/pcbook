all: gen

gen:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative proto/*.proto

clean:
	rm proto/*.go

run: gen
	go run main.go