# Consider it acceptable if lines are removed on CI (because optional
# dependencies are not included), but error if lines are added.
added_lines_count=$(($(git diff --numstat docs | cut -f 1)))
if [ $added_lines_count -ne 0 ]
   then
   echo 'Dependencies documentation must be updated. Please run `make lint-license` locally and commit documentation changes.'
   exit 1
fi
