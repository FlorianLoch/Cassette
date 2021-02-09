#! /usr/bin/env bash
set +e # We want the script to finish as it has to pop the stash even in case the test failed

STASH_NAME="pre-commit-$(date +%s)"
git stash save -q --keep-index --include-untracked $STASH_NAME

make lint && make test
RESULT=$?

LAST_STASH_NAME=$(git stash list | head -n1 | cut -d' ' -f4)
if [ "$LAST_STASH_NAME" = "$STASH_NAME" ]; then
  git stash pop -q
fi

[ $RESULT -ne 0 ] && exit 1
exit 0