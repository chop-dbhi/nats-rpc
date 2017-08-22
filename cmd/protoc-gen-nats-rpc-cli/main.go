package main

import (
	"io/ioutil"
	"log"
	"os"

	natsrpc "github.com/chop-dbhi/nats-rpc"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	req := plugin_go.CodeGeneratorRequest{}
	if err = proto.Unmarshal(data, &req); err != nil {
		log.Fatal(err)
	}

	if len(req.ProtoFile) != 1 {
		log.Fatal("exactly one service proto must be defined")
	}

	pfile := req.ProtoFile[0]

	file, err := natsrpc.ParseFile(pfile, "main.go", tmpl)
	if err != nil {
		log.Fatal(err)
	}

	res := &plugin_go.CodeGeneratorResponse{
		File: []*plugin_go.CodeGeneratorResponse_File{file},
	}

	data, err = proto.Marshal(res)
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.Write(data)
}
