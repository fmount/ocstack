#!/bin/bash

export KUBECONFIG=$(pwd)/kubeconfig

echo "=== Multi-Tool Comparison Test ==="
echo "Starting OCStack with multi-tool comparison query..."
echo

# Create input commands file
cat > /tmp/ocstack_input <<'EOF'
/mcp connect http http://localhost:8080/mcp
Please get both the deployed OpenStack version and the available OpenStack version, then compare them and tell me if a minor update is possible and what the differences are between the versions.
/exit
EOF

echo "Input commands:"
cat /tmp/ocstack_input
echo
echo "=== OCStack Output ==="

# Run ocstack with input and capture output
timeout 120s ./bin/ocstack < /tmp/ocstack_input 2>&1

echo
echo "=== Test Complete ==="

# Clean up
rm -f /tmp/ocstack_input