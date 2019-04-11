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
	go build -o centerms

lint:
	golangci-lint run --no-config --issues-exit-code=0 --deadline=30m \
        --disable-all --enable=deadcode  --enable=gocyclo --enable=golint --enable=varcheck \
        --enable=structcheck --enable=maligned --enable=errcheck --enable=dupl --enable=ineffassign \
        --enable=interfacer --enable=unconvert --enable=goconst --enable=gosec --enable=megacheck

run:
	./centerms

clean:
	rm centerms

dbuild:
	docker build -t centerms .

dlint:
	hadolint Dockerfile

up:
	docker-compose up

down:
	docker-compose down
