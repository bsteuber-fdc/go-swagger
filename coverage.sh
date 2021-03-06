#!/bin/bash

set -e -x

godep go test -v -race ./... | go-junit-report -dir $CIRCLE_TEST_REPORTS/go

# Run test coverage on each subdirectories and merge the coverage profile.
echo "mode: set" > profile.cov
repo_pref="github.com/${CIRCLE_PROJECT_USERNAME-"$(basename `pwd`)"}/${CIRCLE_PROJECT_REPONAME-"$(basename `pwd`)"}/"
# Standard go tooling behavior is to ignore dirs with leading underscores
set -x
for dir in $(go list ./... | grep -v generator)
do
  pth="${dir//*$repo_pref}"
  godep go test -covermode=set -coverprofile=${pth}/profile.tmp $dir
  if [ -f $pth/profile.tmp ]
  then
      cat $pth/profile.tmp | tail -n +2 >> profile.cov
      rm $pth/profile.tmp
  fi
done

set +x
godep go tool cover -func profile.cov
gocov convert profile.cov | gocov report
gocov convert profile.cov | gocov-html > $CIRCLE_ARTIFACTS/coverage-$CIRCLE_BUILD_NUM.html
