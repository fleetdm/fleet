

usage() {
    echo "Usage: $0 [options] (optional|start_version)"
    echo ""
    echo "Options:"
   echo "  -c, --conflicts_resolved The script has been run, had merge conflicts, and those have been resolved."
    echo "  -h, --help             Display this help message and exit"
    echo ""
    echo "Examples:"
    echo "  $0 33499               Update PR 33499"
    echo "  $0 33499 -c            Finish PR 33499 update after conflicts are resolved"
    echo ""
}

conflicts_resolved=false

# Parse long options manually
for arg in "$@"; do
  shift
  case "$arg" in
    "--conflicts_resolved") set -- "$@" "-c" ;;
    "--help") set -- "$@" "-h" ;;
    *)        set -- "$@" "$arg"
  esac
done

# Extract options and their arguments using getopts
while getopts "acdfhgkmno:pqrs:t:uv:w" opt; do
    case "$opt" in
        c) conflicts_resolved=true ;;
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
    if [[ "$conflicts_resolved" == "false" ]]; then
        git checkout main
        git pull origin main
        gh pr checkout $1
        gh pr view $1 --json headRefName | jq -r .headRefName > fu_origin_branch
        git checkout -b $update_branch_name
        git rebase main
        # this is expected to fail with conflicts at this point
        echo "Please fix the merge conflicts and commit locally then rerun with $0 $1 -c"
        exit 0
    else
        git push origin $update_branch_name
        origin_branch=`cat fu_origin_branch`
        gh pr create -f -B $origin_branch
        rm -f fu_origin_branch
        echo "Go merge your PR in w/ an additional approval"
    fi
}

main $@
