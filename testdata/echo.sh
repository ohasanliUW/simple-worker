#!/bin/sh
count=5

for i in $(seq $count -1 0); do
   echo "$i"
   sleep 1
done
