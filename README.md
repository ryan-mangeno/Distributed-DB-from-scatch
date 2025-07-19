# Distributed DB

## Overview
A simple client CLI to connect to a distributed db store running on remote VMs. The client allows storing and retrieving user data via a tcp with a Go node coordinator.

## Setup
Clone the repo and navigate to the project folder.

Create a .env file in the cli/ directory with the following variables:
```bash
HOST_IP_ADDR="<ip-addr>"
HOST_PORT="<port>"
```

## Running the Client
```bash
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r src/cli/requirements.txt
python3 src/cli/client.py
```
## Example Session
```
Connecting to VM at 23.92.34.51:8081
Connected successfully!
Enter command (e.g., 'PUT key value', 'GET key', or type 'exit' to quit): PUT ryan age 1  
Server response: OK: User ryan saved.
Enter command (e.g., 'PUT key value', 'GET key', or type 'exit' to quit): GET ryan
Server response: OK: Found ryan with age 1
Enter command (e.g., 'PUT key value', 'GET key', or type 'exit' to quit): exit
Closing connection.
```