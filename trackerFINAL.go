package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"strings"
	"sync"
)

// map of user id to port number
var users = make(map[string]int)
var keys []string
var list_length int = 3
var fileChunks = map[string]string{
	"simple:1": "This is the content of the example file. It will be split into three chunks.",
	"simple:2": "Each peer will have a subset of these chunks.",
	"simple:3": "Some chunks will overlap between peers."}
var limit int = 0

// var fileChunks = make(map[string]string)
var mu sync.Mutex

// helper functions:

// func getChunksForPeer() []string {
// 	mu.Lock()
// 	defer mu.Unlock()

// 	// Create a slice of chunk keys
// 	chunkKeys := make([]string, 0, len(fileChunks))
// 	for k := range fileChunks {
// 		chunkKeys = append(chunkKeys, k)
// 	}

// 	// Generate a random index
// 	randIndex := rand.IntN(len(chunkKeys))

// 	// Select the chunk at the random index and the next one (wrapping around if necessary)
// 	subset := []string{
// 		chunkKeys[randIndex],
// 	}
// 	return subset
// }

func getChunksForPeer(peerIndex int, totalPeers int) []string {
	mu.Lock()
	defer mu.Unlock()

	// Create a slice of chunk keys
	chunkKeys := make([]string, 0, len(fileChunks))
	for key := range fileChunks {
		chunkKeys = append(chunkKeys, key)
	}

	// Shuffle the chunk keys
	rand.Shuffle(len(chunkKeys), func(i, j int) {
		chunkKeys[i], chunkKeys[j] = chunkKeys[j], chunkKeys[i]
	})

	// Distribute chunks to peers in a round-robin fashion
	var assignedChunks []string
	for i, chunk := range chunkKeys {
		if i%totalPeers == peerIndex {
			assignedChunks = append(assignedChunks, chunk)
		}
	}

	return assignedChunks
}

// hard coded for now
func getUserList() []string {
	if len(keys) <= 3 {
		return keys
	}
	// return list of users
	var subset []string
	var index int = 0
	for index < list_length {
		//shuffle and get a random key
		rand := rand.IntN(len(keys))
		//if !findString(keys[rand], keys) { //this doesn't work bc all the ip addresses are the same, stop the hard code and in the client peer process

		subset = append(subset, keys[rand])
		index++
		//}
	}

	return subset
}

// find the int in the array
func findString(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// when adding a user, add it to the list
func addUser(ip string) { // TODO: are we adding just IP or IP:PORT string?
	users[string(ip)] = 0
	keys = append(keys, string(ip))
}

func removeUser1(ip string) {
	// removes nodes that failed/left the network
	delete(users, ip)
	//remove from the list of keys
	index := indexOf(keys, ip)
	keys = append(keys[:index], keys[index+1:]...)
}

func indexOf(arr []string, val string) int {
	for i, v := range arr {
		if v == val {
			return i
		}
	}
	return -1 // Value not found
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)

	// listen for response from client
	for {

		//sending lists of users
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(err)
			return
		}
		message := strings.TrimSpace(string(buf[:n]))
		if strings.Contains(message, "hii") {
			parts := strings.Split(message, " ")
			if len(parts) == 2 {
				clientAddress := parts[1]
				addUser(clientAddress)
				fmt.Printf("Added client address: %s\n", clientAddress)
				response := "ack!"
				fmt.Fprintf(conn, "%s\n", response)
				fmt.Printf("Sent ack %s: \n", clientAddress)
			}

		}

		// if client request is for list of in-network peers
		// if strings.Contains(strings.TrimRight(string(buf[:n]), "\n"), "list") {

		if strings.Contains(message, "list") {
			// separate client IP and port
			parts := strings.Split(message, " ")
			if len(parts) == 2 {
				clientAddress := parts[0]

				// Send list of users
				data := getUserList()
				if len(data) == 0 {
					log.Println("No users found in the list")
					return
				}
				listMessage := strings.Join(data, ", ")

				//limiting the initialization of the nodes
				allChunks := ""
				if limit < 4 {
					// send file chunks to the peer
					// gets random subset of chunk names
					chunks := getChunksForPeer(0, 4) // TODO: only sending filename right now and not actual contents?

					// send corresponding chunk file contents

					for _, chunk := range chunks {
						chunkContent := fileChunks[chunk]
						message = chunk + "-" + chunkContent
						allChunks = allChunks + message + ","
						if err != nil {
							log.Println(err)
							return
						}
					}
					limit++
				}
				response := listMessage + "\n---\n" + allChunks
				fmt.Fprintf(conn, "%s\n", response)
				fmt.Printf("Sent list of peers and file chunks to %s: %s\n", clientAddress, allChunks)

				//fmt.Printf("Current num of users: %d\n\n", len(keys))

			} else {
				fmt.Println("Invalid message format")
			}
		}

		// }

		//checks
		if strings.Contains(message, "USER FAILED - ") {
			parts := strings.Split(message, " - ")
			removeUser1(parts[1])
		}

		// if client connected with neighbors
		if strings.Contains(strings.TrimRight(string(buf[:n]), "\n"), "connected with neighbors!") {
			return
		}

	}
}

func main() {

	// hard-coded chunks
	// fileChunks["simple:1"] = "This is the content of the example file. It will be split into three chunks."
	// fileChunks["simple:2"] = "Each peer will have a subset of these chunks."
	// fileChunks["simple:3"] = "Some chunks will overlap between peers."

	// TODO: listening on port 8000, change so it's not hardcoded?
	addr, err := net.ResolveTCPAddr("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	fmt.Println("Listening on port 8000")
	for {

		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConnection(conn)
	}
}
