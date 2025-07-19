set -e

echo "Starting service destruction..."

echo "Stopping services..."
sudo systemctl stop storage-engine.service || true
sudo systemctl stop node-coordinator.service || true

echo "  Disabling services (removes auto-start)..."
sudo systemctl disable storage-engine.service || true
sudo systemctl disable node-coordinator.service || true

echo "Removing service files from systemd..."
sudo rm -f /etc/systemd/system/storage-engine.service
sudo rm -f /etc/systemd/system/node-coordinator.service

echo "Reloading systemd daemon to apply changes..."
sudo systemctl daemon-reload

echo ""
echo "Services have been successfully destroyed."