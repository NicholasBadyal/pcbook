FROM golang:alpine
WORKDIR /app
RUN apk add make protoc protobuf-dev git gcc musl-dev
RUN go get -u github.com/golang/protobuf/protoc-gen-go
COPY go.mod .
RUN go mod download
COPY . .
CMD ["make", "server"]