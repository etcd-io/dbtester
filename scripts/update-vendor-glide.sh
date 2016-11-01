#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "scripts/update-vendor-glide.sh" ]]; then
    echo "must be run from repository root"
    exit 255
fi

# update depedency
# 1. edit glide.yaml with version, git SHA
# 2. run ./script-update-vendor-glide.sh
# 3. it automatically detects new git SHA, and vendors updates to cmd/vendor directory
#
# add depedency
# 1. run ./script-update-vendor-glide.sh github.com/USER/PROJECT#^1.0.0
#        OR
#        ./script-update-vendor-glide.sh github.com/USER/PROJECT#9b772b54b3bf0be1eec083c9669766a56332559a
#        ./script-update-vendor-glide.sh golang.org/x/time#711ca1cb87636abec28122ef3bc6a77269d433f3
# 2. make sure glide.yaml and glide.lock are updated

GLIDE_ROOT="$GOPATH/src/github.com/Masterminds/glide"
GLIDE_SHA=21ff6d397ccca910873d8eaabab6a941c364cc70
go get -d -u github.com/Masterminds/glide
pushd "${GLIDE_ROOT}"
	# git reset --hard HEAD
	git reset --hard ${GLIDE_SHA}
	go install
popd

GLIDE_VC_ROOT="$GOPATH/src/github.com/sgotti/glide-vc"
GLIDE_VC_SHA=d96375d23c85287e80296cdf48f9d21c227fa40a
go get -d -u github.com/sgotti/glide-vc
pushd "${GLIDE_VC_ROOT}"
	# git reset --hard HEAD
	git reset --hard ${GLIDE_VC_SHA}
	go install
popd

rm -rf ./vendor


if [ -n "$1" ]; then
	echo "glide get on $(echo $1)"
	glide get --strip-vendor $1
else
	echo "glide update on *"
	glide update --strip-vendor
fi;

glide vc --no-tests --only-code
