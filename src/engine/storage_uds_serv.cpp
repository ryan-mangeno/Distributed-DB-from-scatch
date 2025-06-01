#include <iostream>     
#include <string>      
#include <sys/socket.h> 
#include <sys/un.h>    
#include <unistd.h>     
#include <vector>     
#include <cstring>    


// to add -> fmt::print for logging


// This script is ran on the VM's used in this project


// used as the address for the Unix Domain Socket
// Both server and client must use this exact pat
constexpr const char* SOCKET_PATH = "/tmp/storage_engine.sock";

// function to process a command received from the go coordinator
// For now, it's a placeholder. Later, this will interact with  data store
std::string process_command(const std::string& command) {
    std::cout << "C++ Storage Engine: Received command: \"" << command << "\"" << std::endl;
    // Placeholder: Echo back the command or return a simple success message
    // In  real DB, parse the command (PUT, GET, etc.) and perform actions
    if (command.rfind("PUT", 0) == 0) { 
        return "SUCCESS: Data for '" + command + "' notionally stored.";
    } 
    else if (command.rfind("GET", 0) == 0) { 
        return "DATA: Value for '" + command + "' (not really implemented yet).";
    }
    return "ERROR: Unknown command.";
}

void handle_client_connection(int client_fd) {
    std::cout << "C++ Storage Engine: Client connected (fd: " << client_fd << ")" << std::endl;
    std::vector<char> buffer(1024); // buf for incoming commands

    // read command from coordinator
    ssize_t bytes_received = recv(client_fd, buffer.data(), buffer.size() - 1, 0);

    if (bytes_received > buffer.size() - 1) {
        std::cout << "C++ Storage Engine: Recieved too much data!" << std::endl;
    }
    if (bytes_received > 0) {
        buffer[bytes_received] = '\0'; // null-terminate the received command
        std::string command_str(buffer.data());

        std::string response_str = process_command(command_str);

        // send response back to coordinator
        ssize_t bytes_sent = send(client_fd, response_str.c_str(), response_str.length(), 0);
        if (bytes_sent == -1) {
            perror("C++ Storage Engine: send response failed");
        } 
        else {
            std::cout << "C++ Storage Engine: Response sent." << std::endl;
        }
    } 
    else if (bytes_received == 0) {
        std::cout << "C++ Storage Engine: Client disconnected gracefully." << std::endl;
    } 
    else {
        perror("C++ Storage Engine: recv command failed");
    }

    close(client_fd); // Close this specific client connection
    std::cout << "C++ Storage Engine: Client connection (fd: " << client_fd << ") closed." << std::endl;
}

int main() {
    int server_fd;
    struct sockaddr_un server_addr;

    server_fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (server_fd == -1) {
        perror("C++ Storage Engine: socket creation failed");
        return EXIT_FAILURE;
    }

    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sun_family = AF_UNIX;
    strncpy(server_addr.sun_path, SOCKET_PATH, sizeof(server_addr.sun_path) - 1);
    server_addr.sun_path[sizeof(server_addr.sun_path) - 1] = '\0';

    unlink(SOCKET_PATH); // remove old socket file if it exists, this is neccesary before bind

    if (bind(server_fd, (struct sockaddr*)&server_addr, sizeof(server_addr)) == -1) {
        perror("C++ Storage Engine: bind failed");
        close(server_fd);
        return EXIT_FAILURE;
    }

    if (listen(server_fd, 5) == -1) { // 5 is the backlog for max pending connections
        perror("C++ Storage Engine: listen failed");
        close(server_fd);
        unlink(SOCKET_PATH);
        return EXIT_FAILURE;
    }
    std::cout << "C++ Storage Engine: Listening on UDS path: " << SOCKET_PATH << std::endl;

    // main server loop: continuously accept and handle coordinator connections
    while (true) {
        struct sockaddr_un client_addr; // not strictly needed for processing but good for debug
        socklen_t client_addr_len = sizeof(client_addr);

        // accept() blocks waiting for a UDS client, go coordinator, to connect
        // returns a new file descriptor for this specific client connection
        int client_fd = accept(server_fd, (struct sockaddr*)&client_addr, &client_addr_len);
        if (client_fd == -1) {
            perror("C++ Storage Engine: accept failed");
            // for a more robust server, I might continue or log, rather than exit,
            // unless it's a fatal error with the listening socket itself
            // for simplicity now, let it continue if one accept fails
            continue;
        }
        handle_client_connection(client_fd); // process req
    }

    // cleanup ... (though the while(true) loop means this part is not reached in this simple version)
    // A real server would have a shutdown mechanism ( like signal handler ) to reach here
    close(server_fd);
    unlink(SOCKET_PATH);
    std::cout << "C++ Storage Engine: Shutting down." << std::endl; 

    return EXIT_SUCCESS;
}
