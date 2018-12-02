# go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
# go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
# go get -u github.com/golang/protobuf/protoc-gen-go

gen:
	@echo "generating protobufs..."
	@protoc \
    	--proto_path=${GOPATH}/src \
    	--proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf \
    	    --gogofaster_out=plugins=grpc,Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:./proto \
    	    --govalidators_out=gogoimport=true,Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:./proto \
    	--proto_path=./proto \
    	./proto/*.proto

build:
	go build ./cmd/centerms

clean:
	rm centerms

run:
	docker run -p 50051:50051 centerms

swaggen:
	@cd ./cmd/centerms && \
	swagger generate spec -o ../../swagger.json

swagserv: 
	swagger serve -p=8090 -F=swagger swagger.json
