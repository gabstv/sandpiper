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

./test_all/_build_ $PREVDIR/test_all/config.yml
