# Simple-raft

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

Init(address, nodelist)
- address: of current node
- nodelist: list of nodes to contact on start up

Update(data map[string]string) 
    data - this is what the data of the frame will be set to.