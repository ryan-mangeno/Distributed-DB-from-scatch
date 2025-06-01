package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv" // for .env
)

const (
	// network server constants, listenings for clients
	NETWORK_SERVER_HOST         = "0.0.0.0" // listen on all VM's network interfaces
	NETWORK_SERVER_PORT_DEFAULT = "123"     // Defined in .env or replace here
	NETWORK_SERVER_TYPE         = "tcp"

	// Unix Domain Socket path for storage engine ...
	// This MUST match the SOCKET_PATH in storage_uds_server.cpp
	storageEngineSocketPath = "/tmp/storage_engine.sock"
)

func getListenPort() string {
	// os.Getenv will now pick up variables loaded by godotenv from the .env file
	port := os.Getenv("NODE_TCP_PORT")
	if port == "" {
		port = NETWORK_SERVER_PORT_DEFAULT // Use the default const if NODE_TCP_PORT is not found
		fmt.Printf("Go Node Coordinator: NODE_TCP_PORT not set, using default port %s\n", port)
	} else {
		fmt.Printf("Go Node Coordinator: Using port %s from NODE_TCP_PORT\n", port)
	}
	return port
}

func main() {

	err := godotenv.Load()
	if err != nil {
		// You can choose to log this as a warning or make it fatal if .env is required
		log.Printf("Warning: Error loading .env file: %s. Will rely on existing env vars or defaults.", err)
	}

	fmt.Println("Go Node Coordinator: Starting TCP Server...")

	listenPort := getListenPort()

	listener, err := net.Listen(NETWORK_SERVER_TYPE, NETWORK_SERVER_HOST+":"+listenPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening on TCP: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("Go Node Coordinator: TCP Server listening on %s:%s\n", NETWORK_SERVER_HOST, listenPort)

	for {
		tcpConn, err := listener.Accept() // Accepts incoming TCP connections from remote clients
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting TCP connection: %v\n", err)
			continue // Continue to accept other connections
		}
		// Handle each TCP client connection concurrently using a goroutine.
		go handleNetworkClient(tcpConn)
	}
}

// handleNetworkClient processes messages from a single remote TCP client and interacts with the uds storage
func handleNetworkClient(tcpConn net.Conn) {
	clientAddr := tcpConn.RemoteAddr().String()
	fmt.Printf("Go Node Coordinator: TCP Client connected from %s\n", clientAddr)
	defer func() {
		tcpConn.Close() // ensure TCP connection is closed when this handler exits, gets called before end of handleNetworkClient
		fmt.Printf("Go Node Coordinator: TCP Client %s disconnected\n", clientAddr)
	}()

	tcpReader := bufio.NewReader(tcpConn)

	for { // while loop to handle multiple commands from the same TCP client
		// reading command from the client
		commandFromTcpClient, err := tcpReader.ReadString('\n')
		if err != nil {
			if err == io.EOF { //  way to detect client closing connection
				fmt.Printf("Go Node Coordinator: TCP Client %s closed connection (EOF).\n", clientAddr)
			} else {
				fmt.Fprintf(os.Stderr, "Go Node Coordinator: Error reading from TCP client %s: %v\n", clientAddr, err)
			}
			return // Exit this handler, which will close the TCP connection via defer
		}
		commandFromTcpClient = strings.TrimSpace(commandFromTcpClient) // Remove newline/whitespace
		if commandFromTcpClient == "" {
			continue // Skip empty commands
		}

		fmt.Printf("Go Node Coordinator: Received from TCP client %s: '%s'\n", clientAddr, commandFromTcpClient)

		// Now, act as a UDS client to send this command to the C++ storage engine
		udsResponse, err := sendCommandToStorageEngine(commandFromTcpClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Go Node Coordinator: Error communicating with storage engine for client %s: %v\n", clientAddr, err)
			// Send an error message back to the TCP client
			_, writeErr := tcpConn.Write([]byte("ERROR: Could not process command internally.\n"))
			if writeErr != nil {
				fmt.Fprintf(os.Stderr, "Go Node Coordinator: Error sending error to TCP client %s: %v\n", clientAddr, writeErr)
			}
			continue // Continue processing next command from this TCP client, or let it disconnect
		}

		fmt.Printf("Go Node Coordinator: Received from Storage Engine: '%s'\n", udsResponse)

		// Send the response from the storage engine back to the original TCP client
		_, err = tcpConn.Write([]byte(udsResponse + "\n")) // Add newline for clarity
		if err != nil {
			fmt.Fprintf(os.Stderr, "Go Node Coordinator: Error writing to TCP client %s: %v\n", clientAddr, err)
			return
		}
		fmt.Printf("Go Node Coordinator: Sent response to TCP client %s\n", clientAddr)
	}
}

// sendCommandToStorageEngine connects to the  UDS server, sends a command, and gets a response.
func sendCommandToStorageEngine(command string) (string, error) {
	// net.Dial connects to the UDS path specified.
	udsConn, err := net.Dial("unix", storageEngineSocketPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to UDS storage engine: %w", err)
	}
	defer udsConn.Close() // Ensure UDS connection is closed on exit

	// Send the command
	_, err = udsConn.Write([]byte(command)) // Command does not need newline if  recv doesn't expect it as terminator
	if err != nil {
		return "", fmt.Errorf("failed to send command to UDS storage engine: %w", err)
	}

	// Read the response from the C++ UDS server
	udsBuffer := make([]byte, 2048) // Buffer for the response
	n, err := udsConn.Read(udsBuffer)
	if err != nil {
		return "", fmt.Errorf("failed to read response from UDS storage engine: %w", err)
	}

	return string(udsBuffer[:n]), nil // Convert received bytes to string
}
