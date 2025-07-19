PROJECT_ROOT=$(pwd)

echo " Stopping existing services..."
pkill -f storage_server
pkill -f "go run node_coordinator.go"

# add a small delay to ensure ports are freed 
sleep 2

echo "Pulling latest changes from Git..."
git pull origin main

echo  "Compiling C++ Storage Engine..."
cd "$PROJECT_ROOT/src/engine/"

g++ -o storage_server storage_uds_serv.cpp -std=c++17

# check if compilation was successful
if [ $? -ne 0 ]; then
    echo " C++ compilation failed!"
    cd "$PROJECT_ROOT"
    exit 1
fi

echo " C++ compilation successful."
echo " Starting C++ Storage Engine in the background..."


./storage_server > storage_server.log 2>&1 &



echo " Preparing Go Node Coordinator..."
cd "$PROJECT_ROOT/src/coorindator/"

go mod tidy

echo " Starting Go Node Coordinator in the background..."
#  nohup to ensure the process keeps running.
# logging output to a file inside its own directory.
nohup go run node_coordinator.go > node_coordinator.log 2>&1 &

cd "$PROJECT_ROOT"

sleep 2
echo ">>> Deployment complete. Services should be running."

# Verify that the processes are running
ps aux | grep -E "storage_server|node_coordinator"