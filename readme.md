# Simple-raft

## Description
This is my attempt at a raft consensus implementation.  I wrote this because I needed some form of consensus between the managers that handle what work needs to be done in my HATS environment but didn't need a complicated implementation pulling in dependencies.  For what I am doing being able to keep a map synced between managers with some way to declare who is the source of record was sufficient.

Service discovery is easy and handles the introduction and synching of new nodes into the system no matter which node they introduce themselves to first.

## Processes and Problem Situation Rules
Introduction process:
When a new node is started it will contact a node from the list passed at runtime.  
When a node gets an introduction request that node will either tell the new node who the current leader is so that the new node can introduce itself OR if the node is the current leader it will give the new node all the information it needs to catch up. The leader will then include the new node in future heart beats.

Information that is kept in sync this way are:
- Current list of nodes that get heart beats.
- A data map of strings
- Election number

Multiple leader problem:
If a leader gets a heartbeat sent to it by another leader it will compare the election number from the heart beat with its own election number.  Who ever has the higher election number stays the leader and the other conforms its data to the new leader.

Leader stops sending heartbeats:
After a random time if a node hasn't gotten a heart beat it will hold an election.  It increments its election number and sends an election request to each node it knows about.  They will respond with yes.  After we get a vote from each node we contacted become leader and inform each node.  Start sending heart beats.

## Exposed Functions

Init(address string, nodelist string, updateInterval time.Duration, leaderTimeout time.Duration, electionTimeout time.Duration)
- address: of current node
- nodelist: list of nodes to contact on start up
- updateInterval: the delay between leader updates
- electionTimeout: the delay until elections timeout

Update(data map[string]string) 
    data - this is what the data of the frame will be set to.

State string
- follower, voter, candidate, leader
- Exposed to allow the modification of your application's behavior based on its raft state.