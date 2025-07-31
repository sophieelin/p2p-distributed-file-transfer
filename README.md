# P2P Distributed File Transfer System

A decentralized, peer-to-peer file transfer system enabling direct secure exchange of files across a distributed network.

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


## Getting Started
```bash
# Clone the repository
git clone https://github.com/yourusername/p2p-distributed-file-transfer.git
cd p2p-distributed-file-transfer

# Run the tracker server
go run trackerFINAL.go

# In separate terminals, run as many client peers as you want
go run clientPeerFINAL.go
