# Distributed DB from Scratch

---
**Status: In Active Development**
This project is currently being built  ... rough edges and ongoing changes.
---

## How It Works (Current Two-VM Simulation)

The system is being developed and tested using two virtual machines to simulate a distributed environment. Each VM runs a "node" consisting of:

1.  **C++ Storage Engine:**
    * A dedicated process on each VM responsible for the low-level persistence of data (to be fully implemented).
    * It currently listens on a **Unix Domain Socket (UDS)** for local commands from its Go coordinator.

2.  **Go Node Coordinator:**
    * The primary process on each VM that manages network communication and orchestrates operations.
    * It listens for external client connections on a **TCP/IP socket**.
    * When it receives a command (e.g., from the Python client), it translates this and communicates with its local C++ Storage Engine via the UDS.

3.  **Python Client:**
    * A remote script used to connect to either Node Coordinator (on its VM's IP and TCP port) to send commands and receive responses.

4. **MongoDB:**
    * NoSQL document database used to store and manage data in a flexible, JSON-like format


## Project Setup & Running the Simulation

**Prerequisites:**
* VM's. Preferably atleast two.
* SSH access to your VMs.
* Git installed on your local machine and VMs for cloning, or scp works.
* MongoDB and MongoDB C++ Driver

---
### A. On the Linux VM (Node Setup)

Perform these steps on one of your Linux VMs.

**VM Prerequisites:**

1.  **Update package lists:**
    ```bash
    sudo apt update
    ```
2.  **Install C++ compiler and build tools:**
    ```bash
    sudo apt install build-essential -y
    ```
3.  **Install Go:**
    ```bash
    sudo apt install golang-go -y
    # Verify installation
    go version
    ```

**1. C++ Storage Engine (UDS Server)**

* **Location:** Assume the C++ code (e.g., `storage_uds_serv.cpp`) is in a directory like `~/Distributed-DB-from-scratch/src/cpp-storage-engine/` on the VM.
* **Navigate to the directory:**
    ```bash
    cd ~/Distributed-DB-from-scratch/src/cpp-storage-engine/ 
    ```
* **Compile:**
    ```bash
    g++ -o storage_server storage_uds_serv.cpp -std=c++17
    ```
* **To Run (keep this terminal open):**
    ```bash
    ./storage_server
    ```
    *This server listens on a Unix Domain Socket, typically `/tmp/storage_engine.sock` (as defined in the C++ code).*

* **Downloading MongoDB and its C++ Driver**
    * Follow [this guide](https://www.mongodb.com/docs/manual/tutorial/install-mongodb-on-ubuntu/#std-label-install-mdb-community-ubuntu) for mongodb installation on unbuntu 
    * Follow [this guide](https://www.mongodb.com/docs/languages/cpp/cpp-driver/current/get-started/) for the driver installation

**2. Go Node Coordinator (TCP Server & UDS Client)**

* **Location:** Assume the Go code (e.g., `node_coordinator.go`) is in a directory like `~/Distributed-DB-from-scratch/src/coordinator/` on the VM. This directory should also contain your `go.mod` file.
* **Navigate to the directory:**
    ```bash
    cd ~/Distributed-DB-from-scratch/src/coordinator/
    ```
* **Install Go Dependencies:** If you are using third-party packages like `godotenv`, ensure they are in your `go.mod` file (e.g., by running `go get github.com/joho/godotenv` once in this directory). Then tidy up:
    ```bash
    go mod tidy
    ```
* **Configuration (Port Number):**
    The Go server listens on a TCP port. Configure this by creating a `.env` file in this same Go project directory (`~/Distributed-DB-from-scratch/src/coordinator/`) on the VM:
    **`.env` file contents:**
    ```ini
    NODE_TCP_PORT="<your_tcp_port>" 
    ```
    *(Choose an available port. If this file or variable is not present, the Go script might use a hardcoded default like "<your_tcp_port>").*
* **Firewall:** Open the TCP port on the VM's firewall (e.g., for port <your_tcp_port>):
    ```bash
    sudo ufw allow <your_tcp_port>/tcp
    sudo ufw enable # If not already enabled, ensure SSH is allowed first (sudo ufw allow ssh), don't forget to ensure ssh port is still open
    sudo ufw status
    ```
    *Also ensure any cloud provider firewall/security group rules allow this port.*
* **To Run (in a new, separate terminal on the VM, keep it open):**
    ```bash
    go run node_coordinator.go
    ```
    *This server listens on TCP (e.g., `0.0.0.0:<your_tcp_port>`) for remote clients and communicates with the C++ Storage Engine via UDS.*

---
### B. On Your Local PC (Client Setup)

**Local PC Prerequisites:**

1.  **Python 3:** Ensure it's installed using python3 --version, then cd into ./scripts from the root dir
2.  **Venv Scripts**
    ```bash
    activate_env.bat
    ```
    *(Use `pip3` in the script if `pip` is not being used).*

**3. Python Remote Client**

* **Location:** The Python client code (e.g., `client.py`) is in a directory like `~/projects/Distributed-DB-from-scratch/cli/` on your local machine.
* **Navigate to the directory:**
    ```bash
    cd ~/projects/Distributed-DB-from-scratch/cli/
    ```
* **Configuration:** Create a `.env` file in this same directory (`cli/`) on your local machine:
    **`.env` file contents:**
    ```ini
    HOST_IP_ADDR="YOUR_VM_PUBLIC_IP_ADDRESS"
    HOST_PORT="<your_tcp_port>" 
    ```
    *Replace `YOUR_VM_PUBLIC_IP_ADDRESS` with the actual public IP of your Linux VM.*
    *Ensure `<your_tcp_port>` matches the port your Go Node Coordinator is listening on.*
* **To Run:**
    ```bash
    python client.py
    ```
    *The client will connect to the Go Node Coordinator on your VM. You can then type commands like `GET key` or `PUT key value`.*
