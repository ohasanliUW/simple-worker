#!/bin/sh
count=30

for i in $(seq $count -1 0); do
   echo "$i"
   sleep 1
done
