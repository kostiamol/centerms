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
	go build -o centerms.exe

run:
	go build -o centerms.exe
	./centerms.exe

clean:
	rm centerms.exe
