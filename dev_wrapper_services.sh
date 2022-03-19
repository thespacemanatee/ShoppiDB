#!/bin/bash

# Start the first process
/redis/redis-6.2.6/src/redis-server --save 60 1 --dir /redis/volume &
  
# Start the second process
reflex -d none -c /usr/local/etc/reflex.conf
  
# Wait for any process to exit
wait -n
  
# Exit with status of process that exited first
exit $?