# P2P Distributed File Transfer System
A decentralized, peer-to-peer (P2P) file transfer system built with Go, designed to enable direct and secure exchange of files across a distributed network without relying on a central server. This system allows multiple peers to act as both clients and servers, facilitating efficient file sharing in real time. Each peer can upload, download, and request files from any other connected peer in the network.

## Features
- Peer discovery
- Decentralized architecture

## Components
### Server 
- Handles peer discovery and routing
- Tracks active connections and number of peers in the distributed network
- Does not store or contain any files for decentralized design

### Client
- Sends and receives files directly with peers
- Manages file transfers and connections
- Initiates requests for files and peer discovery
- Implements acknowledgment (ACK) messages and retransmits lost packets to ensure reliable data transfer
- Each node stores a portion of files in the distributed network


## Setup Instructions
```bash
# Clone the repository
git clone https://github.com/yourusername/p2p-distributed-file-transfer.git
cd p2p-distributed-file-transfer

# Run the tracker server
go run trackerFINAL.go

# In separate terminals, run as many client peers as you want
go run clientPeerFINAL.go
