#!/bin/bash
MODE=count
GOVER=`which gover`
REPORT="go tool cover -html=gover.coverprofile"
if [[ $1 == "travis-ci" ]]; then
  GOVER=$HOME/gopath/bin/gover
  GOVERALLS=$HOME/gopath/bin/goveralls
  REPORT="$GOVERALLS -service=travis-ci -repotoken=${COVERALLS_TOKEN} -coverprofile=gover.coverprofile"
fi

go test ./... -covermode=$MODE -coverprofile=gover.coverprofile

if [[ ! -z "$GOVER" ]]; then
  $GOVER
fi

$REPORT 
