package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// HTTPClient interface
type httpClient interface {
	Post(url string, contentType string, body io.Reader) (*http.Response, error)
}

//ServerInterface implement this interface to customize raft behavior
type ServerInterface interface {
	CommitLogs([]string) error        // Called when follower is told to commit logs
	GetLogsToPush() ([]string, error) // Called when leader is told to get logs it needs to push
	StateChange(string) error         // Called when state changes
}

type Node struct {
	Address           string
	LastRequestFailed bool
	server            ServerInterface
}

type Frame struct {
	CurrentLeader  Node
	ElectionNumber int
	Nodes          []Node
	LastUpdated    time.Time
	Logs           []string
}

var me Node
var votes = 0

var myFrame *Frame
var State = "follower"
var address = ""

var client httpClient
var raftServer *http.Server

var updateInterval time.Duration
var leaderTimeout time.Duration
var electionTimeout time.Duration

func Init(addr string, serverList string, serverInterface ServerInterface, updateI time.Duration, leaderT time.Duration, electionT time.Duration) {
	if client == nil {
		client = &http.Client{}
	}

	updateInterval = updateI
	leaderTimeout = leaderT
	electionTimeout = electionT

	myFrame = &Frame{}
	address = addr

	//Some routes
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", handleDefault)
		mux.HandleFunc("/introduce", handleIntroduction)
		mux.HandleFunc("/vote", handleVote)
		mux.HandleFunc("/election", handleElection)
		mux.HandleFunc("/lead", handleLeader)
		mux.HandleFunc("/frame", handleFrame)
		raftServer = &http.Server{Addr: addr, Handler: mux}
		err := raftServer.ListenAndServe()
		log.WithError(err).Error("Error listening")
	}()

	me = Node{Address: address, server: serverInterface}

	//Introduce myself to all the servers in the list.
	servers := strings.Split(serverList, ",")
	for _, v := range servers {
		if v != "" {
			introduce(v)
		} else {
			//No one else in this pool so be the leader by default.
			myFrame.CurrentLeader = me
			myFrame.Nodes = append(myFrame.Nodes, me)
			State = "leader"
			me.server.StateChange(State)
		}
	}
}

func Update() {
	log.Info("State: ", State)
	if State == "leader" {
		logs, err := me.server.GetLogsToPush()
		if err != nil {
			log.WithError(err).Error("Error getting logs to push")
			return
		}

		myFrame.Logs = logs
		myFrame.LastUpdated = time.Now()
		for i := range myFrame.Nodes {
			if myFrame.Nodes[i].Address != me.Address {
				sendFrame(myFrame.Nodes[i])
			}
		}
	}

	if State == "follower" {
		//If it's been too long since a frame update then start an election
		if time.Now().Sub(myFrame.LastUpdated) > leaderTimeout {
			State = "candidate"
			me.server.StateChange(State)
			myFrame.ElectionNumber++
			votes = 1 //Vote for myself.
			log.Info("Starting election ", myFrame.ElectionNumber)

			for i := range myFrame.Nodes {
				if myFrame.Nodes[i].Address != me.Address {
					proposeElection(myFrame.Nodes[i])
				}
			}
		}
	}

	if State == "candidate" {
		time.Sleep(electionTimeout)
		if votes == len(myFrame.Nodes) {
			//Won the election.  Become leader.
			State = "leader"
			me.server.StateChange(State)
			for i := range myFrame.Nodes {
				if myFrame.Nodes[i].Address != me.Address {
					becomeLeader(myFrame.Nodes[i])
				}
			}
		} else {
			State = "follower"
			me.server.StateChange(State)
		}
	}
	time.Sleep(updateInterval)
}

func Stop() {
	if raftServer != nil {
		raftServer.Shutdown(context.TODO())
	}
}

func introduce(target string) error {
	log.Info("Introducing myself to ", target)
	b, err := json.Marshal(&IntroductionRequest{Address: address})
	if err != nil {
		log.WithError(err).Error("Failed marshaling intro")
		return err
	}

	resp, err := client.Post("http://"+target+"/introduce", "application/json", bytes.NewBuffer(b))
	if err != nil {
		fmt.Println(err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("Introduction gave non-200 status back")
	}
	return nil
}

func proposeElection(target Node) error {
	log.Info("Proposing election ", target)
	b, err := json.Marshal(&IntroductionRequest{Address: address, ElectionNumber: myFrame.ElectionNumber})
	if err != nil {
		log.WithError(err).Error("Failed marshaling intro")
		votes++ //Corrupt politics. Count their vote anyway.
		return err
	}

	resp, err := http.Post("http://"+target.Address+"/election", "application/json", bytes.NewBuffer(b))
	if err != nil {
		fmt.Println(err)
		votes++ //Corrupt politics. Count their vote anyway.
		return err
	}

	if resp.StatusCode != http.StatusOK {
		votes++ //Corrupt politics. Count their vote anyway.
		return errors.New("Introduction gave non-200 status back")
	}
	return nil
}

//Become the leader of the target Node.  Sends a request to /lead of the Node with current frame.
func becomeLeader(target Node) error {
	log.Info("Becoming the leader of ", target)
	myFrame.CurrentLeader = me
	myFrame.LastUpdated = time.Now()
	b, err := json.Marshal(&myFrame)
	if err != nil {
		log.WithError(err).Error("Failed marshaling intro")
		return err
	}

	resp, err := http.Post("http://"+target.Address+"/lead", "application/json", bytes.NewBuffer(b))
	if err != nil {
		fmt.Println(err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("Introduction gave non-200 status back")
	}
	return nil
}

func sendFrame(target Node) error {
	b, err := json.Marshal(&myFrame)
	if err != nil {
		log.WithError(err).Error("Failed marshaling intro")
		return err
	}

	resp, err := http.Post("http://"+target.Address+"/frame", "application/json", bytes.NewBuffer(b))
	if err != nil {
		fmt.Println(err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("Introduction gave non-200 status back")
	}
	return nil
}

//IntroductionRequest Used to introduce Node to pool.
type IntroductionRequest struct {
	Address        string
	ElectionNumber int
}

func handleDefault(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "", 404)
}

func handleIntroduction(w http.ResponseWriter, r *http.Request) {
	log.Info("Incoming introduction")
	if State == "leader" {
		log.Info("I'm a leader, become the leader of this follower")
		var intro IntroductionRequest
		err := json.NewDecoder(r.Body).Decode(&intro)
		if err != nil {
			log.WithError(err).Error("Failed parsing introduction request")
			http.Error(w, "Failed parsing introduction request", 500)
			return
		}

		//Add the Node if we dont' know them already.
		newNode := Node{Address: intro.Address}
		found := false
		for i := range myFrame.Nodes {
			if myFrame.Nodes[i] == newNode {
				found = true
			}
		}
		if !found {
			log.Info("New follower!")
			myFrame.Nodes = append(myFrame.Nodes, newNode)
			fmt.Println(myFrame.Nodes)
		}

		becomeLeader(newNode)
	} else {
		//Follows forward introductions to the leader.
		log.Info("Forwarding request to leader")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = http.Post("http://"+myFrame.CurrentLeader.Address+"/introduce", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Println(err)
		}
	}
}

func handleLeader(w http.ResponseWriter, r *http.Request) {
	log.Info("New leader!")
	var frame *Frame
	err := json.NewDecoder(r.Body).Decode(&frame)
	if err != nil {
		log.WithError(err).Error("Failed parsing frame request")
		http.Error(w, "Failed parsing frame request", 500)
		return
	}

	if State == "follower" || State == "voter" || State == "candidate" {
		myFrame = frame
		State = "follower"
		me.server.StateChange(State)
	}
	if State == "leader" {
		//Stop being leader, other guy is more recent
		if frame.ElectionNumber > myFrame.ElectionNumber {
			myFrame = frame
			State = "follower"
			me.server.StateChange(State)
		} else {
			for i := range myFrame.Nodes {
				if myFrame.Nodes[i].Address != me.Address {
					becomeLeader(myFrame.Nodes[i])
				}
			}
		}
	}
}

func handleFrame(w http.ResponseWriter, r *http.Request) {
	log.Info("Got Frame!")
	var frame *Frame

	err := json.NewDecoder(r.Body).Decode(&frame)
	if err != nil {
		log.WithError(err).Error("Failed parsing frame request")
		http.Error(w, "Failed parsing frame request", 500)
		return
	}

	if State == "follower" {
		myFrame = frame
		me.server.CommitLogs(myFrame.Logs)
	} else if State == "leader" {
		if frame.ElectionNumber > myFrame.ElectionNumber {
			State = "follower"
			me.server.StateChange(State)
			myFrame = frame
		}
	}
}

func handleVote(w http.ResponseWriter, r *http.Request) {
	var voter IntroductionRequest
	err := json.NewDecoder(r.Body).Decode(&voter)
	if err != nil {
		log.WithError(err).Error("Failed parsing introduction request")
		http.Error(w, "Failed parsing introduction request", 500)
		return
	}

	votes++
}

func handleElection(w http.ResponseWriter, r *http.Request) {
	State = "voter"
	me.server.StateChange(State)
	var candidate IntroductionRequest
	err := json.NewDecoder(r.Body).Decode(&candidate)
	if err != nil {
		log.WithError(err).Error("Failed parsing introduction request")
		http.Error(w, "Failed parsing introduction request", 500)
		return
	}

	if candidate.ElectionNumber > myFrame.ElectionNumber {
		myFrame.ElectionNumber = candidate.ElectionNumber
		log.Info("Voting for ", candidate.Address)
		b, err := json.Marshal(&IntroductionRequest{Address: address})
		if err != nil {
			log.WithError(err).Error("Failed marshaling intro")
			http.Error(w, "Failed marshaling intro", 500)
			return
		}

		resp, err := http.Post("http://"+candidate.Address+"/vote", "application/json", bytes.NewBuffer(b))
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed posting vote", 500)
			return
		}

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Candidate failed to get post", 500)
		}
	}
}
