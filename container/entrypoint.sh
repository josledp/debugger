#!/bin/bash
if [ -z "$CONTAINER_ID" ]; then
  echo "CONTAINER_ID env var empty!"
  exit 1
fi

ID=`echo $CONTAINER_ID|cut -f3 -d/`

pid=`docker inspect $ID -f "{{.State.Pid}}"`

nsenter -t $pid -p -u -i -n --preserve-credentials

