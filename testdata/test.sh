#!/bin/bash

IFS=$'\n'
output=$(cat $1)
header=$(echo "$output" | awk '/### BEGIN COMPGEN INFO/ {p=1;next}; /### END COMPGEN INFO/ {p=0}; p')
opts=$(echo "$output" | awk 'BEGIN{p=1} /### BEGIN COMPGEN INFO/ {p=0;next}; /### END COMPGEN INFO/ {p=1;next}; p')

set_mode() {
  case $1 in
    "filedir" )
      echo "setting mode"
      ;;
  esac

  echo "unknown option:" $1 "-"
}

while read line
do

  read foo key value <<<$(IFS=$':'; echo $line)
  echo "k/v" $key $value
  case $key in
    "mode" )
      set_mode $value
      ;;
  esac
done <<< $header
echo "$opts"
