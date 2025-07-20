package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

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

var (
	nodeTCPPort     string
	nodeRole        string // 'primary'or 'secondary'
	peerNodeAddr    string // address of other nodes ( one single other node in this scenario )
	primaryNodeAddr string // IP address of the primary node (used by secondary for security)
)

func main() {
	loadConfig()

	fmt.Printf("Go Node Coordinator: Starting as %s node...\n", nodeRole)

	listener, err := net.Listen(NETWORK_SERVER_TYPE, net.JoinHostPort(NETWORK_SERVER_HOST, nodeTCPPort))
	if err != nil {
		log.Fatalf("Error listening on TCP: %v\n", err)
	}
	defer listener.Close()
	fmt.Printf("Go Node Coordinator: TCP Server listening on %s:%s\n", NETWORK_SERVER_HOST, nodeTCPPort)

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting TCP connection: %v\n", err)
			continue
		}
		go handleNetworkClient(tcpConn) // this can be expensive when there are many users, but is a workaround for now
	}
}

func handleNetworkClient(tcpConn net.Conn) {
	clientAddr := tcpConn.RemoteAddr().String()
	clientIP, _, _ := net.SplitHostPort(clientAddr)

	fmt.Printf("Go Node Coordinator: TCP Client connected from %s\n", clientAddr)
	defer func() {
		tcpConn.Close()
		fmt.Printf("Go Node Coordinator: TCP Client %s disconnected\n", clientAddr)
	}()

	tcpReader := bufio.NewReader(tcpConn)

	for {
		commandFromClient, err := tcpReader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from TCP client %s: %v\n", clientAddr, err)
			}
			return
		}
		commandFromClient = strings.TrimSpace(commandFromClient)
		if commandFromClient == "" {
			continue
		}

		fmt.Printf("Go Node Coordinator: Received from %s: '%s'\n", clientAddr, commandFromClient)

		// distributed logit
		isWriteCommand := strings.HasPrefix(strings.ToUpper(commandFromClient), "PUT")
		isFromPrimary := (nodeRole == "SECONDARY" && clientIP == primaryNodeAddr)

		if nodeRole == "SECONDARY" && isWriteCommand && !isFromPrimary {
			fmt.Println("Go Node Coordinator: Rejecting write command on secondary node from non-primary client.")
			tcpConn.Write([]byte("ERROR: Write operations are only allowed on the primary node.\n"))
			continue
		}

		localResponse, err := sendCommandToStorageEngine(commandFromClient)
		if err != nil {
			log.Printf("Error from local storage engine for client %s: %v\n", clientAddr, err)
			tcpConn.Write([]byte("ERROR: Could not process command internally.\n"))
			continue
		}

		if nodeRole == "PRIMARY" && isWriteCommand && strings.HasPrefix(localResponse, "OK") {
			fmt.Printf("Go Node Coordinator: Replicating write command to secondary at %s\n", peerNodeAddr)
			err := replicateCommandToSecondary(commandFromClient)
			if err != nil {
				log.Printf("FATAL: Failed to replicate command to secondary: %v. Data is now inconsistent.", err)
				localResponse = "ERROR: Write succeeded locally but failed to replicate to secondary."
			}
		}

		_, err = tcpConn.Write([]byte(localResponse + "\n"))
		if err != nil {
			log.Printf("Error writing to TCP client %s: %v\n", clientAddr, err)
			return
		}
	}
}

func replicateCommandToSecondary(command string) error {
	conn, err := net.DialTimeout("tcp", peerNodeAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("could not connect to secondary node: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(command + "\n"))
	if err != nil {
		return fmt.Errorf("could not send command to secondary: %w", err)
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("did not receive confirmation from secondary: %w", err)
	}

	fmt.Printf("Go Node Coordinator: Received replication confirmation: '%s'\n", strings.TrimSpace(response))
	if !strings.HasPrefix(response, "OK") {
		return fmt.Errorf("secondary node returned an error: %s", response)
	}
	return nil
}

// uds communication
func sendCommandToStorageEngine(command string) (string, error) {
	udsConn, err := net.Dial("unix", storageEngineSocketPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to UDS storage engine: %w", err)
	}
	defer udsConn.Close()

	_, err = udsConn.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("failed to send command to UDS storage engine: %w", err)
	}

	udsBuffer := make([]byte, 2048)
	n, err := udsConn.Read(udsBuffer)
	if err != nil {
		return "", fmt.Errorf("failed to read response from UDS storage engine: %w", err)
	}
	return string(udsBuffer[:n]), nil
}

func loadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Could not load .env file. Relying on environment variables.")
	}

	// Load the port, falling back to the default constant
	nodeTCPPort = os.Getenv("NODE_TCP_PORT")
	if nodeTCPPort == "" {
		nodeTCPPort = NETWORK_SERVER_PORT_DEFAULT
		log.Printf("Warning: NODE_TCP_PORT not set, using default port %s\n", nodeTCPPort)
	}

	// load the role for distributed logic
	nodeRole = os.Getenv("NODE_ROLE")
	if nodeRole == "" {
		log.Fatal("FATAL: NODE_ROLE not set. Must be 'PRIMARY' or 'SECONDARY'.")
	}

	// load peer addresses based on the role
	if nodeRole == "PRIMARY" {
		peerNodeAddr = os.Getenv("SECONDARY_NODE_ADDR")
		if peerNodeAddr == "" {
			log.Fatal("FATAL: PRIMARY node requires SECONDARY_NODE_ADDR to be set.")
		}
	} else if nodeRole == "SECONDARY" {
		primaryNodeIP, _, err := net.SplitHostPort(os.Getenv("PRIMARY_NODE_ADDR"))
		if err != nil || primaryNodeIP == "" {
			log.Fatal("FATAL: SECONDARY node requires a valid PRIMARY_NODE_ADDR to be set.")
		}
		primaryNodeAddr = primaryNodeIP
	}
}
