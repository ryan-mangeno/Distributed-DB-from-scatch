# Distributed DB

## Overview
A simple client CLI to connect to a distributed db store running on remote VMs. The client allows storing and retrieving user data via a tcp with a Go node coordinator.

## Setup
Fork the repo, clone that fork, then cd into Distributed-DB-from-Scratch

Create a .env file in the ```.src/cli/``` directory with the following variables:
```bash
#./src/cli/.env
HOST_IP_ADDR="<ip-addr>"
HOST_PORT="<port>"
```

*Optional for CI/CD*
You will also need to make a repository secret, where the ```deploy.yml``` and ```deploy.sh``` pulls from, which will be accessible from your forked repository page on github -> settings -> Secrets and Variables -> Actions -> New Repository Secret
Then make one named ```NODE_TCP_PORT``` with your desired port number.

You will also need runners for each instance you have running to pull changes, which is accessible at forked repo page -> settings -> actions -> runners -> New self-hosted runner -> then follow instructions ... repeat for each instance or you can be fancy and automate the setup for vm's with ansible.

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