package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var neighbors []string
var portAndIP string
var ips []string
var fileChunks []string
var neighborConns = make(map[string]net.Conn)
var chunksMissing = []string{}             //chunks we DON'T have in format of filename+chunk(ie. fileName1)
var currentFiles = make(map[string]string) //all the current files +chunk it has (ie. fileName1), filename : actual file
var trackerConn net.Conn
var clientAddress string
var mu sync.Mutex
var sharedResource int
var inputChannel = make(chan string)
var mutex sync.Mutex

/*
	{ATTEMPT TO CONNECT WITH NEIGHBORS}

peer sends out request for file to its neighbors
  - iterates through list of neighbors sent from tracker and tries to connect with each
  - stops once it has 3 neighbors
  - if neighbor acknowledges connection, add to list of neighbors
  - if neighbor does not acknowledge connection, send message to tracker to remove from list of peers
*/
func handleNeighborConnections(clientAddress string, neighbors []string) {
	var wg sync.WaitGroup

	for _, neighborIP := range neighbors { // TODO: only connect to 3 neighbors
		neighborIP = strings.TrimSpace(neighborIP)
		if neighborIP != clientAddress { //TODO: don't allow peer to connect to itself
			fmt.Printf("connecting with %s\n", neighborIP)
			// increment waitGroup counter so we wait before proceeding
			wg.Add(1)
			// goroutine to connect to a single neighbor
			// fmt.Println("neighborIP format:\n", neighborIP)
			go func(neighborIP string) {
				defer wg.Done()
				requestNeighbor(clientAddress, neighborIP)
			}(neighborIP)
		} //else {
		// 	fmt.Printf("Skipping connection to self: %s\n", neighborIP)
		// }
	}

	wg.Wait()
}

// a single connection request between one peer and another peer
func requestNeighbor(clientAddress string, neighborIP string) {
	fmt.Printf("req")
	conn, err := net.Dial("tcp", neighborIP)
	if err != nil {
		log.Println("Error connecting to neighbor:", err)
		return
	}
	// defer conn.Close()
	// send message to neighbor
	_, err = fmt.Fprintf(conn, "NEIGHBORS! "+clientAddress+"\n")
	if err != nil {
		log.Println("Error sending message to neighbor:", err)
		return
	}

	// Read acknowledgment from neighbor
	ack, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Println("Error reading acknowledgment from neighbor:", err)
		return
	}
	if strings.TrimSpace(ack) == "Acknowledged" {
		fmt.Printf("Neighbor %s acknowledged the connection\n", neighborIP)
		neighbors = append(neighbors, neighborIP)
		neighborConns[neighborIP] = conn
		fmt.Printf("My list of neighbors %s\n", neighbors)
	} else {
		fmt.Printf("Neighbor %s did not acknowledge the connection\n", neighborIP)

		message := "USER FAILED - " + neighborIP
		_, err = trackerConn.Write([]byte(message))
		if err != nil {
			log.Fatal("Error sending message to tracker for failure node:", err)
		}
		fmt.Println("Message sent for failure node:", message)
	}
}
func main() {
	// manual user input for IP+PORT
	var addr string
	fmt.Print("Please input your IP address and port in the format IP:PORT : ")
	fmt.Scan(&addr)
	clientAddress = strings.TrimSpace(addr)
	IP_PORT := strings.Split(clientAddress, ":")
	if len(IP_PORT) != 2 {
		fmt.Println("Invalid input format. Please use the format IP:PORT.")
		return // TODO: make it ask for the IP address and port again
	}

	trackerAddress := "192.168.1.166:8000" //TODO: change to tester's IP address
	conn, err := net.Dial("tcp", trackerAddress)
	trackerConn = conn
	if err != nil {
		fmt.Println("Error connecting to tracker:", err.Error())
		os.Exit(1)
		// log.Fatal(err)
	}
	fmt.Println("New Peer!\n")
	defer conn.Close()

	fmt.Fprintf(conn, "hii %s\n", clientAddress)
	fmt.Println("1. sent to tracker ")
	buf := make([]byte, 1024)
	value, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error receiving from tracker:", err.Error())
		os.Exit(1)
	}
	trackerResponse := string(buf[:value])
	fmt.Println("1. Received response from tracker:", trackerResponse)

	go listener()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Press Enter to start the server and begin listening...")
	reader.ReadString('\n')

	// ask tracker for list of neighbors + send client IP to tracker
	fmt.Fprintf(conn, "%s list\n", clientAddress)
	fmt.Println("1. sent to tracker ")
	buf = make([]byte, 1024)
	value, err = conn.Read(buf)
	if err != nil {
		fmt.Println("Error receiving from tracker:", err.Error())
		os.Exit(1)
	}
	trackerResponse = string(buf[:value])
	fmt.Println("2. Received response from tracker: %s", trackerResponse)

	// Split the response into neighbors and file chunks
	parts := strings.Split(trackerResponse, "\n---\n")

	// Parse neighbors + attempt to connect
	neighborsList := parts[0]
	potentialNeighbors := strings.Split(strings.TrimSpace(neighborsList), ",")
	for _, n := range potentialNeighbors {
		trimmedn := strings.TrimSpace(n)
		if trimmedn != clientAddress {
			connectToIP(trimmedn)
		}
	}
	fmt.Printf("3. received list from tracker: %v\n", potentialNeighbors)

	// Parse file chunks + add to its list
	chunksList := parts[1]
	fmt.Printf("Chunks list: %s\n", chunksList)
	chunks := strings.Split(strings.TrimSpace(chunksList), ",")
	for _, chunk := range chunks {
		parts := strings.SplitN(chunk, "-", 2)
		if len(parts) == 2 {
			currentFiles[parts[0]] = parts[1] // TODO: part after "=" should be contents of that chunk
		}
	}

	fmt.Printf("4. File chunks received: %v\n", currentFiles)
	fmt.Printf("neighbors %v", neighbors)
	//wg.Wait
	for _, ip := range neighbors {

		go func(addr string) {
			if !inNeighbor(addr) {
				connectToIP(addr)
			}
		}(ip)

	}

	go readUserInput()
	for {
		select {
		case userInput := <-inputChannel:
			fmt.Printf("User input received: %s\n", userInput)
			i := strings.TrimSpace(userInput)
			if i == "simple" {
				personal(i)
			}
			if userInput == "exit" {
				fmt.Println("Exiting...")
				return
			}

			for _, conn := range neighborConns {
				go handlePeerConnection(conn)
			}
		}
	}

	//fmt.Println("All connection attempts completed")
	//go handleNeighborConnections(clientAddress, potentialNeighbors)
	select {}
}

func connectToIP(addr string) {

	conn, err := net.Dial("tcp", addr)

	//conn, err := net.Dial("tcp", addr)

	if err != nil {
		log.Printf("Failed to connect to %s: %v\n", addr, err)
		// Notify tracker about connection failure
		return
	}

	// Send neighbor request
	_, err = conn.Write([]byte("NEIGHBORS! " + clientAddress + "\n"))
	if err != nil {
		log.Printf("Error sending neighbor request to %s: %v\n", addr, err)
		return
	}

	// Read acknowledgment
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Error reading response from %s: %v\n", addr, err)
		return
	}

	if strings.TrimSpace(response) == "Acknowledged" {
		log.Printf("Successfully connected to neighbor: %s\n", addr)
		addNeighbor(addr, conn)
		fmt.Printf("neighbors: %v", neighbors)
		mu.Lock()
		neighborConns[addr] = conn
		mu.Unlock()
	}
}
func listener() {
	listener, err := net.Listen("tcp", clientAddress)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Listening on %s", clientAddress)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handlePeerConnection(conn)
	}
}
func handlePeerConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(err)
			return
		}
		message := string(buf[:n])
		fmt.Printf("Received message from peer: %s\n", message)

		// Handle connection establishment request from the peer
		if strings.Contains(message, "NEIGHBORS! ") {
			parts := strings.Split(message, " ")
			// Respond with acknowledgment
			_, err = conn.Write([]byte("Acknowledged\n"))
			if err != nil {
				log.Println(err)
				return
			}
			addNeighbor(parts[1], conn)
			fmt.Printf("neighbors: %v", neighbors)
			fmt.Printf("connected!")

		}

		// Handle file request from the peer

		if strings.Contains(message, "GET -") {
			// check if peer has file and send, 4 items in message
			parts := strings.Split(message, " - ")
			if len(parts) != 4 {
				fmt.Printf("Invalid message format: %s\n", message)
				continue
			}
			fmt.Printf("parts: %s\n", parts)
			// trim parts
			filename := strings.TrimSpace(parts[1])
			chunkname := strings.TrimSpace(parts[2])
			num := strings.TrimSpace(parts[3])
			// fmt.Printf("parts[1]: %s\n", filename)
			// fmt.Printf("parts[2]: %s\n", chunkname)
			// fmt.Printf("parts[3]: %s\n", numStr)
			numStr, err := strconv.Atoi(num)
			if err != nil {
				log.Fatalf("Error converting string to int: %v", err)
			}
			requestFile(filename, chunkname, numStr)
		}

		if strings.Contains(message, "SENDING FILE - ") {
			parts := strings.Split(message, " - ")
			currentFiles[parts[1]] = parts[2]
			deleteChunk(parts[1])
		}

	}
}
func requestFile(filename string, reqUserIP string, TTL int) {

	fileChunks := strings.Split(filename, ".")
	missingChunks := ""
	for index, chunk := range fileChunks {
		if value, inList := currentFiles[chunk]; inList {
			conn, err := net.Dial("tcp", reqUserIP)
			// if user connection fails, remove neighbor from list of neighbors
			if err != nil {
				log.Println("error connecting to neighbor\n", err)
				//deleteUser(reqUserIP)
				return
			}
			defer conn.Close()
			//send the content
			filechunkMsg := "SENDING FILE - " + filename + " - " + value
			_, err1 := conn.Write([]byte(filechunkMsg))
			if err1 != nil {
				log.Println(err)
				return
			}
		} else {
			if index == 0 {
				missingChunks = missingChunks + chunk
			} else {
				missingChunks = missingChunks + "." + chunk
			}
		}
	}

	// TTL ran out
	if TTL == 0 {
		fmt.Printf("TTL ran out for %s\n", filename)
		//send a msg to the original user that it runs out
		conn, err := net.Dial("tcp", reqUserIP)
		filechunkMsg := "TTL ran out for " + filename
		_, err1 := conn.Write([]byte(filechunkMsg))
		if err1 != nil {
			log.Println(err)
			return
		}
		defer conn.Close()
		return
	}

	//split filenames into two
	// filenameParts := strings.Split(filename, ".")
	// getRequest := "GET -"
	// for index, chunk := range filenameParts {
	// 	// first line is for testing
	// 	fmt.Printf("Chunk %d: %s\n", index, chunk)
	// 	if index == 0 {
	// 		getRequest = getRequest + chunk
	// 	} else {
	// 		getRequest = getRequest + "." + chunk
	// 	}
	// }
	fmt.Printf("missingChunks %s: \n", missingChunks)

	fmt.Printf("%s does not exist in the map\n", missingChunks)
	fmt.Printf("sending requests to neighbors\n")
	//fmt.Printf("%v\n", neighborConns)
	for _, neighborIP := range neighbors {
		//if neighbors != reqUserIP {
		currConn := neighborConns[neighborIP]
		newTTL := TTL - 1
		str := strconv.Itoa(newTTL)
		mutex.Lock()
		defer mutex.Unlock()
		// time.Sleep(time.Duration(chunkIndex*10000) * time.Millisecond)
		// getRequest := "GET - " + filenameParts[1] + "." + filenameParts[2] + " - " + reqUserIP + " - " + str
		getRequest := "GET - " + missingChunks + " - " + reqUserIP + " - " + str
		fmt.Printf("getRequest: %s\n", getRequest)
		_, err := currConn.Write([]byte(getRequest))
		if err != nil {
			log.Println("sending requests to all neighbor", err)
			return
		}
		//	}
	}
	// }
}

// deletes the user from the list
func deleteUser(userIP string) {
	if userIP != clientAddress {
		//delete from the connection variables
		for neighbor := range neighborConns {
			if neighbor == userIP {
				delete(neighborConns, neighbor)
			}
		}
		//delete from the neighbors list of keys
		for index, neighbor := range neighbors {
			if neighbor == userIP {
				neighbors = append(neighbors[:index], neighbors[index+1:]...)
			}
		}
		//send to tracker to delete
		message := "USER FAILED - " + userIP
		_, err1 := trackerConn.Write([]byte(message))
		if err1 != nil {
			log.Fatal("Error sending message:", err1)
		}
		fmt.Println("Message sent:", message)
		log.Println("removing this node from my neighbor list:\n")
	}
}
func determineNeededChunks() {
	temp := []string{"simple:1", "simple:2", "simple:3"}

	for _, filename := range temp {
		if _, inList := currentFiles[filename]; !inList {
			// If the chunk is not in currentFiles, append it to temp
			chunksMissing = append(chunksMissing, filename)
		}
	}
}

func deleteChunk(chunkname string) {
	var temp []string
	for _, value := range chunksMissing {
		if value != chunkname {
			temp = append(temp, value)
		}
	}
	chunksMissing = temp
}

func addNeighbor(ip string, conn net.Conn) {
	if ip != clientAddress {
		if !inNeighbor(ip) {
			neighbors = append(neighbors, ip)
			neighborConns[ip] = conn

		}
	}
}

func inNeighbor(ip string) bool {
	for _, neighbor := range neighbors {
		if ip == neighbor {
			return true
		}
	}
	return false

}
func readUserInput() {
	fmt.Printf("What file do you want? Avaliable files: (simple)")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if scanner.Scan() {
			userInput := scanner.Text()
			inputChannel <- userInput // Send input to the channel
		}
		if err := scanner.Err(); err != nil {
			log.Println("Error reading input:", err)
		}

	}
}
func personal(onlyFileName string) {
	determineNeededChunks()
	fmt.Printf("Chunks missing: %v\n", chunksMissing)

	var wg sync.WaitGroup
	wg.Add(len(chunksMissing))
	concatMsg := ""
	fmt.Printf("neighbors: %v\n", neighbors)

	for _, chunk := range chunksMissing {
		// first line is for testing
		// fmt.Printf("Chunk %d: %s\n", index, chunk)
		concatMsg = concatMsg + "." + chunk
	}

	// fmt.Printf("Concatenated message: %s\n", concatMsg)
	// requestFile(concatMsg, clientAddress, 2)
	go func() {
		defer wg.Done()
		// requestFile(concatMsg, clientAddress, 2) // TLL runs out when TTL = 2
		requestFile(concatMsg, clientAddress, 5)
	}()

	wg.Wait()
	fmt.Println("All chunk requests completed")
	fmt.Printf("Printing simple file:")
	for _, value := range chunksMissing {
		fmt.Printf(currentFiles[value])
	}
}
