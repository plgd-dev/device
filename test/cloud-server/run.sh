#!/usr/bin/env bash
set -e

umask 0000

args=( "$@" )
for ((i=0; i < $#; i++)) ;do
  if [[ "${args[$i]}" == "-test.coverprofile="* ]]; then
    coverprofile=${args[$i]#"-test.coverprofile="}
    break
  fi
done

/usr/local/bin/device.client.test $@ -test.coverprofile=/tmp/device.client.test.coverage1.txt
/usr/local/bin/device.client.core.test $@ -test.coverprofile=/tmp/device.client.core.test.coverage2.txt

if [ ! -z "$coverprofile" ]; then
  cp /tmp/device.client.test.coverage1.txt $coverprofile
  tail -n +2 /tmp/device.client.core.test.coverage2.txt >> $coverprofile
fi
