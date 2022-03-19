#!/bin/bash

# Start the first process
/src/redis-server --daemonize yes &
  
# Start the second process
reflex -d none -c /usr/local/etc/reflex.conf
  
# Wait for any process to exit
wait -n
  
# Exit with status of process that exited first
exit $?