#!/bin/sh
count=$1

for i in $(seq $count -1 1); do
   echo "stdout $i"
   >&2 echo "stderr $i"
   sleep 1
done
