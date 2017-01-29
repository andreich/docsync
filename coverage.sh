#!/bin/bash
MODE=count
GOVER=`which gover`
REPORT="go tool cover -html=gover.coverprofile"
if [[ $1 == "travis-ci" ]]; then
  GOVER=$HOME/gopath/bin/gover
  GOVERALLS=$HOME/gopath/bin/goveralls
  REPORT="$GOVERALLS -coverprofile=gover.coverprofile"
fi

for I in `go list ./...`; do
  go test $I -covermode=$MODE -coverprofile=$(basename $I).coverprofile;
done

$GOVER
$REPORT
