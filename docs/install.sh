#!/bin/bash

# Install the systemd service
echo "Installing Port Monitor Service..."

# Copy the service file to systemd directory
sudo cp ./port-monitor.service /etc/systemd/system/

# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable the service to start on boot
sudo systemctl enable port-monitor.service

# Start the service
sudo systemctl start port-monitor.service

echo "Port Monitor Service installed and started!"
echo ""
echo "To check service status: sudo systemctl status port-monitor.service"
echo "To stop the service: sudo systemctl stop port-monitor.service"
echo "To restart the service: sudo systemctl restart port-monitor.service"
echo ""
echo "Access the dashboard at http://localhost:7337"