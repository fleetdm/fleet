#!/bin/bash

sed -i "s/placeholder/${ENROLL_SECRET}/g" /etc/default/orbit
export $(cat /etc/default/orbit | xargs)

while true; do
	echo "Starting orbit..."
	/opt/orbit/bin/orbit/orbit
	echo "orbit exit code: $?"
	sleep 5
done
