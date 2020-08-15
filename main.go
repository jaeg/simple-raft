package main

import (
	"flag"
	"strconv"

	"github.com/jaeg/simple-raft/raft"
)

var serverList = flag.String("servers", "", "Comma separated list of server address in pool")
var port = flag.String("port", "7777", "Port")

var data map[string]string

func main() {
	data = make(map[string]string)
	flag.Parse()
	raft.Init("127.0.0.1:"+*port, *serverList)

	// Keep raft updating in a go proc
	go func() {
		for {
			raft.Update(data)
		}
	}()

	// Do normal service stuff based on our raft state.
	var i int64
	for {
		if raft.State == "leader" {
			data["iteration"] = strconv.Itoa(int(i))
			i++
		} else if raft.State == "follower" {
			i, _ = strconv.ParseInt(data["iteration"], 0, 64)
		}
	}
}
