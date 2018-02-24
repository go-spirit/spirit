.PHONY: all proto

all:

proto:
	protoc -I=${PWD}/protocol/proto --go_out=${PWD}/protocol ${PWD}/protocol/proto/protocol.proto