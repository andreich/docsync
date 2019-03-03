#!/bin/bash
MODE=count
PROFILE=gover.coverprofile
REPORT="go tool cover -html=${PROFILE}"
if [[ $1 == "travis-ci" ]]; then
  GOVERALLS=$HOME/gopath/bin/goveralls
  REPORT="$GOVERALLS -service=travis-ci -repotoken=${COVERALLS_TOKEN} -coverprofile=${PROFILE}"
fi

rm ${PROFILE}
go test ./... -covermode=$MODE -coverprofile=${PROFILE}

$REPORT 
ls -l ${PROFILE}
cat ${PROFILE}
