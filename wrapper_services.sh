#!/bin/bash

# Start the first process
redis-server --daemonize yes --save 60 1 --dir /redis/volume &
  
# Start the second process
/app/shoppidb
  
# Wait for any process to exit
wait -n
  
# Exit with status of process that exited first
exit $?