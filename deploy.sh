set -e

echo "[+] Checking and installing build-essential if needed..."
sudo apt-get update -y
sudo apt-get install -y build-essential libsqlite3-dev

# if go not installed
if ! command -v go &> /dev/null; then
  echo "Go not found in PATH. Checking /usr/local/go/bin..."
  if [ -x /usr/local/go/bin/go ]; then
    echo "Go binary found in /usr/local/go/bin. Adding to PATH..."
  else
    echo "Go not installed. Installing Go 1.24.0..."
    cd /tmp
    wget -q https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
    cd -
  fi
else
  echo "Go is already installed: $(go version)"
fi

export GO_PATH=$(command -v go)
echo "Go binary located at: $GO_PATH"

PROJECT_ROOT=$(pwd)

echo "Cleaning up old socket file..."
sudo rm -f /tmp/storage_engine.sock

echo "Stopping and disabling any existing services before update..."
# stop the services if they are running
sudo systemctl stop storage-engine.service || true
sudo systemctl stop node-coordinator.service || true
# Disable the services to remove auto-start symlinks. This effectively "destroys" the old service definition
sudo systemctl disable storage-engine.service || true
sudo systemctl disable node-coordinator.service || true

# add a small delay to ensure ports are freed 
sleep 3

echo "Pulling latest changes from Git..."
git pull origin main

echo  "Compiling C++ Storage Engine..."
cd "$PROJECT_ROOT/src/engine/"

g++ -o storage_server storage_uds_serv.cpp -std=c++17 -lsqlite3

if [ $? -ne 0 ]; then
    echo " C++ compilation failed!"
    cd "$PROJECT_ROOT"
    exit 1
fi

echo " C++ compilation successful."

echo " Preparing Go Node Coordinator..."
cd "$PROJECT_ROOT/src/coorindator/"


echo "Creating distributed .env file..."
if [ -z "$PRIMARY_IP_SECRET" ] || [ -z "$SECONDARY_IP_SECRET" ] || [ -z "$PORT_FROM_SECRET" ]; then
    echo "FATAL: Required IP or Port secrets are not set in the workflow file."
    exit 1
fi

# get the primary IP address of the current VM
LOCAL_IP=$(hostname -I | awk '{print $1}')

# determine the role by comparing the local IP to the secrets
if [ "$LOCAL_IP" == "$PRIMARY_IP_SECRET" ]; then
    echo "Configuring as PRIMARY node."
    echo "NODE_ROLE=PRIMARY" > .env
    echo "SECONDARY_NODE_ADDR=${SECONDARY_IP_SECRET}:${PORT_FROM_SECRET}" >> .env
elif [ "$LOCAL_IP" == "$SECONDARY_IP_SECRET" ]; then
    echo "Configuring as SECONDARY node"
    echo "NODE_ROLE=SECONDARY" > .env
    echo "PRIMARY_NODE_ADDR=${PRIMARY_IP_SECRET}:${PORT_FROM_SECRET}" >> .env
else
    echo "FATAL: This VM's IP does not match PRIMARY_IP_SECRET or SECONDARY_IP_SECRET."
    exit 1
fi
echo "NODE_TCP_PORT=$PORT_FROM_SECRET" >> .env


go mod tidy

echo " Starting Go Node Coordinator in the background..."



# Get the absolute path to the project for the service files
CPP_EXEC_PATH="$PROJECT_ROOT/src/engine/storage_server"
GO_PROJECT_PATH="$PROJECT_ROOT/src/coorindator/"

# create the service file for the C++ engine
sudo bash -c "cat > /etc/systemd/system/storage-engine.service" <<EOF
[Unit]
Description=Distributed DB C++ Storage Engine
After=network.target

[Service]
Type=simple
ExecStart=$CPP_EXEC_PATH
WorkingDirectory=$PROJECT_ROOT/src/engine/
Restart=on-failure
RestartSec=5
User=runner
StandardOutput=journal
StandardError=journal
PrivateTmp=false  # <-- IMPORTANT
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

[Install]
WantedBy=multi-user.target
EOF

# create the service file for the Go coordinator
sudo bash -c "cat > /etc/systemd/system/node-coordinator.service" <<EOF
[Unit]
Description=Distributed DB Go Node Coordinator
After=storage-engine.service
BindsTo=storage-engine.service

[Service]
Type=simple
WorkingDirectory=$GO_PROJECT_PATH
ExecStart=$GO_PATH run node_coordinator.go
Restart=on-failure
RestartSec=5
User=runner
Environment="NODE_TCP_PORT=$PORT_FROM_SECRET"

[Install]
WantedBy=multi-user.target
EOF


echo "Reloading systemd and starting services..."
sudo systemctl daemon-reload
sudo systemctl enable storage-engine.service
sudo systemctl enable node-coordinator.service
sudo systemctl restart storage-engine.service
sudo systemctl restart node-coordinator.service

echo "Deployment complete. Services are now managed by systemd"
echo "Checking service status:"
sudo systemctl status storage-engine.service --no-pager
sudo systemctl status node-coordinator.service --no-pager
