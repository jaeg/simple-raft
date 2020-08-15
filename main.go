package main

import (
	"flag"

	"github.com/jaeg/simple-raft/raft"
)

var serverList = flag.String("servers", "", "Comma separated list of server address in pool")
var port = flag.String("port", "7777", "Port")

var data map[string]string

func main() {
	data = make(map[string]string)
	flag.Parse()
	raft.Init("127.0.0.1:"+*port, *serverList)

	for {
		raft.Update(data)
	}
}
