

usage() {
    echo "Usage: $0 [options] (optional|start_version)"
    echo ""
    echo "Options:"
    echo "  -h, --help             Display this help message and exit"
    echo ""
    echo "Examples:"
    echo "  $0 33499               Update PR 33499"
    echo ""
}

conflicts_resolved=false

# Parse long options manually
for arg in "$@"; do
  shift
  case "$arg" in
    "--help") set -- "$@" "-h" ;;
    *)        set -- "$@" "$arg"
  esac
done

# Extract options and their arguments using getopts
while getopts "h" opt; do
    case "$opt" in
        h) usage; exit 0 ;;
        ?) usage; exit 1 ;;
    esac
done

check_gh() {
    gh repo set-default
}

main() {
    check_gh
    if [[ "$1" == "" ]]; then
        echo "PR number required"
        exit 1
    fi
    update_branch_name="$USER-$1-mu"
    current_branch=`git rev-parse --abbrev-ref HEAD`
    if [[ "$current_branch" == "$update_branch_name" ]]; then
        conflicts_resolved=true
    fi
    if [[ "$conflicts_resolved" == "false" ]]; then
        git checkout main
        git pull origin main
        gh pr checkout $1
        gh pr view $1 --json headRefName | jq -r .headRefName > fu_origin_branch
        git checkout -b $update_branch_name
        git rebase main
        # this is expected to fail with conflicts at this point
	
        echo "Please fix the merge conflicts, git add and commit locally then rerun with $0 $1"

        exit 0
    else
        git push origin $update_branch_name
        origin_branch=`cat fu_origin_branch`
        gh pr create -f -B $origin_branch -t "Rebase main into long lived feature branch" -b "This is j
ust pulling in commits from main that have landed since this branch was created. Just need a thumb from
codeowners to pull in these changes. Thank you for your quick responses!"
        rm -f fu_origin_branch
        echo "Go merge your PR in w/ an additional approval"
        # put us back on feature branch
        gh pr checkout $1
    fi
}

main $@
