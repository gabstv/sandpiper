#!/bin/bash

set -e

#
PREVDIR=$(pwd)
#
mkdir -p ../tmp
cd ../cmd/sandpiper
go build -o _test_all_
mv _test_all_ $PREVDIR/test_all/_build_

cd $PREVDIR

# config hosts
T0=$(./util/manage-etc-hosts.sh -e alpha.sandpiper)
if [ "$T0" == "0" ]; then
    ./util/manage-etc-hosts.sh -a alpha.sandpiper
fi
T0=$(./util/manage-etc-hosts.sh -e bravo.sandpiper)
if [ "$T0" == "0" ]; then
    ./util/manage-etc-hosts.sh -a bravo.sandpiper
fi
T0=$(./util/manage-etc-hosts.sh -e main.sandpiper)
if [ "$T0" == "0" ]; then
    ./util/manage-etc-hosts.sh -a main.sandpiper
fi

# start alpha
go run $GOPATH/src/github.com/gabstv/sandpiper/test/websites/alpha/main.go &
# start bravo
go run $GOPATH/src/github.com/gabstv/sandpiper/test/websites/bravo/main.go &

./test_all/_build_ $PREVDIR/test_all/config.yml