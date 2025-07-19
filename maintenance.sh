# This script cleans up disk space on the self-hosted runner VM.
set -e

echo "Starting VM Maintenance..."

echo  "Cleaning apt package cache..."
sudo apt-get clean

echo "Cleaning old system logs (keeping last 7 days)..."
sudo journalctl --vacuum-time=2d

echo " Cleaning up GitHub Actions runner work directory..."
if [ -d /home/runner/actions-runner/_work ]; then
    sudo rm -rf /home/runner/actions-runner/_work/*
    echo "   - Runner _work directory cleaned."
else
    echo "   - Runner _work directory not found."
fi

echo " Cleaning up GitHub Actions runner diagnostic logs..."
if [ -d /home/runner/actions-runner/_diag ]; then
    sudo rm -rf /home/runner/actions-runner/_diag/*
    echo "   - Runner _diag directory cleaned."
else
    echo "   - Runner _diag directory not found."
fi

echo "VM Maintenance Complete. Current disk space:"
df -h /
