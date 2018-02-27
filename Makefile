.PHONY: all proto

all:

proto-fbp:
	protoc -I=${PWD}/worker/fbp/protocol/proto --go_out=${PWD}/worker/fbp/protocol ${PWD}/worker/fbp/protocol/proto/protocol.proto