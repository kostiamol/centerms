# go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
# go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
# go get -u github.com/golang/protobuf/protoc-gen-go

build:
	protoc  -I api/pb/ api/pb/api.proto \
		-I/usr/local/include \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:api/pb \
		--swagger_out=logtostderr=true:api/pb

	# GOOS=linux GOARCH=amd64 go build ./cmd/centerms
	# docker build -t centerms .
	
run: 
	docker run -p 50051:50051 centerms
