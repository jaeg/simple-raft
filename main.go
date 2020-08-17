package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/jaeg/simple-raft/raft"
)

var serverList = flag.String("servers", "", "Comma separated list of server address in pool")
var port = flag.String("port", "7777", "Port")

//Server to implement raft
type Server struct {
}

//CommitLogs implement commit logs trigger
func (s Server) CommitLogs(logs []string) error {
	fmt.Println("Commit Logs")
	fmt.Println(logs)
	return nil
}

//GetLogsToPush Implement this to determine what logs to push to followers
func (s Server) GetLogsToPush() ([]string, error) {
	fmt.Println("Push Logs")
	logsOut := make([]string, 0)
	for i := 0; i < 10; i++ {
		log := LogMessage{Operation: "Set", Value: i}
		jsonLog, err := json.Marshal(log)
		if err != nil {
			return nil, err
		}
		logsOut = append(logsOut, string(jsonLog))
	}
	return logsOut, nil
}

//StateChange handles a raft state change
func (s Server) StateChange(state string) error {
	fmt.Println("State Change ")
	return nil
}

type LogMessage struct {
	Operation string
	Value     int
}

func main() {
	flag.Parse()
	server := Server{}
	raft.Init("127.0.0.1:"+*port, *serverList, server, time.Second/2, time.Second, time.Second)

	// Keep raft updating in a go proc
	go func() {
		for {
			raft.Update()
		}
	}()

	// Do normal service stuff based on our raft state.
	var i int64
	for {
		if raft.State == "leader" {
			i++
		} else if raft.State == "follower" {
		}
	}
}
