#!/bin/bash

# Start the first process
redis-server --daemonize yes &
  
# Start the second process
./shoppidb
  
# Wait for any process to exit
wait -n
  
# Exit with status of process that exited first
exit $?