#!/usr/bin/env bash

# Usage: mark_rita_db_finished.sh DB_Name [mongo parameters]

# This script is helpful if you have stopped ipfix-rita and would
# like to analyze the results it has collected so far

require_util () {
  #Returns true if all binaries listed as parameters exist somewhere in the path, False if one or more missing.
  while [ -n "$1" ]; do
    if ! type -path "$1" >/dev/null 2>/dev/null ; then
      echo Missing utility "$1". Please install it. >&2
      return 1        #False, app is not available.
    fi
    shift
  done
  return 0        #True, app is there.
}

require_util mongo

get_mongo_cmd() {
  echo "db.getSiblingDB(\"MetaDatabase\").databases.update(
    {name: \"$1\"},
    {\"\$set\": {
      import_finished: true,
    }}
  )"
}

if [ -z "$1" -o "$1" = "-h" -o "$1" = "--help" ]; then
  echo "Usage: mark_rita_db_finished.sh DB_Name [mongo parameters]"
  echo ""
  echo "Marks a RITA database ready for analysis."
  exit 1
fi

db=$1
shift

get_mongo_cmd "$db" | mongo "$@"
