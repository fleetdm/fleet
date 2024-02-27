#!/usr/bin/env bash

#                                                                                                     
#                                                      ,::;;,                                         
#                                                  ,:;;;:,,;:                                         
#                                              ,::;;+: ,:;;:                                          
#                                            ,;;;:,,++;;:,                                            
#                                           :;;;::++:,                                                
#                                           ++:,  ,;:                                                 
#            ,,                            ,;:     ::+;:::  ,,,                                       
#         ,:;::;:                          :::     ,:,;:,*;;;;+,,,,::,,,                              
#         ;:    :+:                       ,;:,      ,:++;;:;**;++::::::;;;;:,                         
#         ,;  ,::,:;:,                    :,:        :,;;+*+;, ::        :;:;;:,                      
#          ,;:;,    :;:,                 ,;:*:       ,::?*:    :: ,,,,  ,:::::;++;;:::::,,            
#           :;        :;:,               ,;+;       ,:;;;   ,::;;;::::::;;:,,,,:+,,,,,:::;;::,        
#            ;,         ,;:,              :+,    ,:;;,  ;::;:,,,::      ::    :;          ,;?+        
#            ,;,          ,;:,           ,;:   ,:;:,    ,+;,    ::,     ::,,:;:    ,,,::;;;+;         
#             ,;            ,+;,         :::,:;++,        :;;;;;;;;;;;;;;+;;;::;;;;;::,,:;:,          
#              ::          ,;:,;;,      ,;;;:,:+, ,,:          ,,:::;;;;;;**;,,,,    ,:;:,            
#               ::       ,;:,   ,;;,   ,;;:,     :;;:,    ,,::::,,:;:,,  ;+*+    ,:;;;,               
#                ;,    ,::,       ,:;;;:,        ::;,:,,::::,  ,:;:      ,++:,::;;:,                  
#                ,;,  :;,        ,:;++,          ::+:;:,,    :;:,        ,:;;;:,                      
#                 ,;,;:       ,:;++:,          ,:;++;,    ,;+:,      ,:;;;:,                          
#                  :;     ,,:;+;:,          ,:::,,,,,,, :;;+;    ,:;;::,                              
#                   ;:   :;+;:,          ,:::,    ,;+;;;:::;;,:;;::,                                  
#                   ;;,;*+:,,:,       ,:::,      ,;, ,;: :,++:,,                                      
#                  :;:*;:   :;,     ,::,        ,:,  ::, ::;                                          
#                ,:+:,      ::    ,::          ,:,  ;:  ,;;                                           
#               ,;,   ,,,:  ;;,,,;; ,;;       ,:, ,;:,:+;, ,,,,,                                      
#                ;::;;;::*;:;;;:::;;++,      ,:, ,+;;::;+;;;:::;+,                                    
#                +;,    :?;;:::,   ,+,      ,:, :;,,,:;;:,     :;                                     
#               ,+,     :+,,,,,;,,,;;      ,:, ;+;;;:;;      ,;;                                      
#               :;     :;      ,::;+,     ,:,,;:,,,  ::  ,,:;;,                                       
#               ;:    :;          ;;     ,:  ++;+;; ,;+;;;::,                                         
#              ,;,   ::          :;:    ,:  ::,,,::+?*+;:                                             
#              :;   ::          ;::,   ,: ,;:,:;;:,;;:*+;                                             
#             ,;:  ::          ,: :   ,: :;,;:,    :+;+,                                              
#             ,;, ;:           ;,,:  ,:,;+ ,:     ,;+:                                                
#             ;;;;:           ,: :, ,;:*++;;;+++**+:                                                  
#             ;:,,            ;, ;:;+::,, ,:;+:::+;                                                   
#                            ,:,;;:,  ,,:;;::::;;:                                                    
#                            +;:,,,:;;;;;;;;::,                                                       
#                           ;?+;;;:,:;;::,                                                            
#                           ,,,+;,,:;                                                                 
#                               :;::,                                                                 
#                                                                                                     
#                                                                                                     
#  /$$$$$$$$ /$$       /$$$$$$$$ /$$$$$$$$ /$$$$$$$$       /$$$$$$$   /$$$$$$  /$$$$$$$$ /$$$$$$  /$$   /$$
# | $$_____/| $$      | $$_____/| $$_____/|__  $$__/      | $$__  $$ /$$__  $$|__  $$__//$$__  $$| $$  | $$
# | $$      | $$      | $$      | $$         | $$         | $$  \ $$| $$  \ $$   | $$  | $$  \__/| $$  | $$
# | $$$$$   | $$      | $$$$$   | $$$$$      | $$         | $$$$$$$/| $$$$$$$$   | $$  | $$      | $$$$$$$$
# | $$__/   | $$      | $$__/   | $$__/      | $$         | $$____/ | $$__  $$   | $$  | $$      | $$__  $$
# | $$      | $$      | $$      | $$         | $$         | $$      | $$  | $$   | $$  | $$    $$| $$  | $$
# | $$      | $$$$$$$$| $$$$$$$$| $$$$$$$$   | $$         | $$      | $$  | $$   | $$  |  $$$$$$/| $$  | $$
# |__/      |________/|________/|________/   |__/         |__/      |__/  |__/   |__/   \______/ |__/  |__/
#
# /$$$$$$$  /$$$$$$$$ /$$       /$$$$$$$$  /$$$$$$   /$$$$$$  /$$$$$$$$ /$$$$$$$
# | $$__  $$| $$_____/| $$      | $$_____/ /$$__  $$ /$$__  $$| $$_____/| $$__  $$
# | $$  \ $$| $$      | $$      | $$      | $$  \ $$| $$  \__/| $$      | $$  \ $$
# | $$$$$$$/| $$$$$   | $$      | $$$$$   | $$$$$$$$|  $$$$$$ | $$$$$   | $$$$$$$/
# | $$__  $$| $$__/   | $$      | $$__/   | $$__  $$ \____  $$| $$__/   | $$__  $$
# | $$  \ $$| $$      | $$      | $$      | $$  | $$ /$$  \ $$| $$      | $$  \ $$
# | $$  | $$| $$$$$$$$| $$$$$$$$| $$$$$$$$| $$  | $$|  $$$$$$/| $$$$$$$$| $$  | $$
# |__/  |__/|________/|________/|________/|__/  |__/ \______/ |________/|__/  |__/
#

usage() {
    echo "Usage: $0 [options] (optional|start_version)"
    echo ""
    echo "Options:"
    echo "  -d, --dry_run          Perform a trial run with no changes made"
    echo "  -f, --force            Skipp all confirmations"
    echo "  -h, --help             Display this help message and exit"
    echo "  -m, --minor            Increment to a minor version instead of patch (Required if including non-bugs"
    echo "  -o, --open_api_key     Set the Open API key for calling out to ChatGPT"
    echo "  -s, --start_version    Set the target starting version (can also be the first positional arg) for the release, defaults to latest release on github"
    echo "  -t, --target_date      Set the target date for the release, defaults to today if not provided"
    echo "  -v, --target_version   Set the target version for the release"
    echo ""
    echo "Environment Variables:"
    echo "  OPEN_API_KEY           Open API key used for fallback if not provided via -o or --open-api-key option"
    echo ""
    echo "Examples:"
    echo "  $0 -d                  Dry run the script"
    echo "  $0 -m -v 4.45.1        Set a minor release targeting version 4.45.1"
    echo "  $0 --target_version 4.45.1 --open_api_key examplekey"
    echo ""
}

# Usage example: Run a command and show spinner for n seconds
# Replace `sleep 5` with your command
# sleep 5 & show_spinner 5
show_spinner() {
    local pid=$!
    local delay=0.1
    local spinstr='/-\|'
    local elapsedTime=0
    local maxTime=$1

    printf "Processing "
    while [ $elapsedTime -lt $maxTime ]; do
        local temp=${spinstr#?}
        printf "%c" "$spinstr"
        local spinstr=$temp${spinstr%"$temp"}
        sleep $delay
        printf "\b"
        elapsedTime=$((elapsedTime+1))
    done

    printf "\nDone.\n"
}

check_grep() {
    # Check if `grep` supports the `-P` option by using it in a no-op search.
    # Redirecting stderr to /dev/null to suppress error messages in case `-P` is not supported.
    if echo "" | grep -P "" >/dev/null 2>&1; then
        return
    else
        # Now check if `ggrep` is available.
        if command -v ggrep >/dev/null 2>&1; then
            return
        else
            echo "Please install latest grep with `brew install grep`"
            exit 1
        fi
    fi
}

check_required_binaries() {
    local missing_counter=0
    # List of required binaries used in the script
    local required_binaries=("jq" "gh" "git" "curl" "awk" "sed" "make")

    for bin in "${required_binaries[@]}"; do
        if ! command -v "$bin" &> /dev/null; then
            echo "Error: Required binary '$bin' is not installed." >&2
            missing_counter=$((missing_counter + 1))
        fi
    done

    if [ $missing_counter -ne 0 ]; then
        echo "Error: $missing_counter required binary(ies) are missing. Install them before running this script." >&2
        exit 1
    fi
    check_grep
}

validate_and_format_date() {
    local input_date="$1"
    local formatted_date
    local correct_format="%b %d, %Y" # e.g., Jan 01, 2024

    # Try to convert input_date to the correct format
    formatted_date=$(date -d "$input_date" +"$correct_format" 2>/dev/null)

    if [ $? -ne 0 ]; then
        # date conversion failed
        echo "Error: Incorrect date format. Expected format example: $correct_format (e.g., Jan 01, 2024)" >&2
        exit 1
    else
        # Check if the formatted date matches the expected date format
        if ! date -d "$formatted_date" +"$correct_format" &>/dev/null; then
            # This means the formatted date does not match our correct format
            echo "Error: Incorrect date format after conversion. Expected format example: $correct_format (e.g., Jan 01, 2024)" >&2
            exit 1
        fi
    fi

    # If we reached here, the date is valid and correctly formatted
    target_date="$formatted_date" # Update the target_date with the formatted date
    echo "Validated and formatted date: $target_date"
}

# Validate we have all commands required to perform this script
check_required_binaries

# Initialize variables for the options
dry_run=false
force=false
minor=false
open_api_key=""
start_version=""
target_date=""
target_version=""

# Parse long options manually
for arg in "$@"; do
  shift
  case "$arg" in
    "--dry-run") set -- "$@" "-d" ;;
    "--force") set -- "$@" "-f" ;;
    "--help") set -- "$@" "-h" ;;
    "--minor") set -- "$@" "-m" ;;
    "--open_api_key") set -- "$@" "-o" ;;
    "--start_version") set -- "$@" "-s" ;;
    "--target_date") set -- "$@" "-t" ;;
    "--target_version") set -- "$@" "-v" ;;
    *)        set -- "$@" "$arg"
  esac
done

# Extract options and their arguments using getopts
while getopts "dfhmo:s:t:v:" opt; do
    case "$opt" in
        d) dry_run=true ;;
        f) force=true ;;
        h) usage; exit 0 ;;
        m) minor=true ;;
        o) open_api_key=$OPTARG ;;
        s) start_version=$OPTARG ;;
        t) target_date=$OPTARG ;;
        v) target_version=$OPTARG ;;
        ?) usage; exit 1 ;;
    esac
done

# Shift off the options and optional --
shift $((OPTIND -1))

# Function to determine the best grep variant to use
determine_grep_command() {
    # Check if `ggrep` is available
    if command -v ggrep >/dev/null 2>&1; then
        echo "ggrep"  # Use GNU grep if available
    elif echo "" | grep -P "" >/dev/null 2>&1; then
        echo "grep"  # Use grep if it supports the -P option
    else
        echo "grep"  # Default to grep if ggrep is not available and -P is not supported
        # Note: You might want to handle the lack of -P support differently here
    fi
}

# Assign the best grep variant to a variable
GREP_CMD=$(determine_grep_command)

# Now you can use the $dry_run variable to see if the option was set
if $dry_run; then
    echo "Dry run mode enabled."
fi

# Check for OPEN_API_KEY environment variable if no key was provided through command-line options
if [ -z "$open_api_key" ]; then
    if [ -n "$OPEN_API_KEY" ]; then
        open_api_key=$OPEN_API_KEY
    else
        echo "Error: No open API key provided. Set the key via -o/--open-api-key option or OPEN_API_KEY environment variable." >&2
        exit 1
    fi
fi

if [[ "$target_date" != "" ]]; then
    validate_and_format_date $target_date
fi

# ex v4.43.0
if [ -z "$start_version" ]; then
    if [[ "$1" == "" ]]; then
        # grab latest draft excluding test version 9.99.9
        draft=`gh release list | $GREP_CMD Draft | $GREP_CMD -v 9.99.9`
        if [[ "$draft" != "" ]]; then
            start_version=`echo $draft | awk '{print $1}' | cut -d '-' -f2`
        else
            start_version=`gh release list | $GREP_CMD Latest | awk '{print $1}' | cut -d '-' -f2`
        fi
    else
        start_version="$1"
    fi
fi

if [[ $start_version != v* ]]; then
    start_version=`echo "v$start_version"`
fi

if [[ "$target_version" != "" ]]; then
    if [[ $target_version != v* ]]; then
        target_version=`echo "v$target_version"`
    fi
    next_ver=$target_version
else
    if [[ "$minor" == "true" ]]; then
        next_ver=$(echo $start_version | awk -F. '{print $1"."($2+1)".0"}')
    else
        next_ver=$(echo $start_version | awk -F. '{print $1"."$2"."($3+1)}')
    fi
fi

start_ver_tag=fleet-$start_version



echo "Patch release from $start_version to $next_ver"
if [ "$force" = "false" ]; then
    read -r -p "If this is correct confirm yes to continue? [y/N] " response
    case "$response" in
        [yY][eE][sS]|[yY])
            echo
            ;;
        *)
            exit 1
            ;;
    esac
fi
target_milestone="${next_ver:1}"
target_milestone_number=`gh api repos/:owner/:repo/milestones | jq -r ".[] | select(.title==\"$target_milestone\") | .number"`


if [[ "$target_milestone_number" == "" ]]; then
    echo "Missing milestone $target_milestone, Please create one and tie tickets to the milestone to continue"
    exit 1
fi
echo "Found milestone $target_milestone with number $target_milestone_number"

if [ "$dry_run" = "false" ]; then
    git fetch
fi

# TODO Fail if not found
if [ "$dry_run" = "false" ]; then
    git checkout $start_ver_tag
else
    echo "DRYRUN: Would have checked out starting tag $start_ver_tag"
fi

target_patch_branch="patch-fleet-$next_ver"

local_exists=`git branch | $GREP_CMD $target_patch_branch`

if [ "$dry_run" = "false" ]; then
    if [[ $local_exists != "" ]]; then
        # Clear previous
        git branch -D $target_patch_branch
    fi
    git checkout -b $target_patch_branch
else
    echo "DRYRUN: Would have cleared / checked out new branch $target_patch_branch"
fi


total_prs=()

issue_list=`gh issue list --search 'milestone:"'"$target_milestone"'"' --json number | jq -r '.[] | .number'`
if [[ "$issue_list" == "" ]]; then
    echo "Milestone $target_milestone has no target issues, please tie tickets to the milestone to continue"
fi
echo "Issue list for new patch $next_ver"
echo $issue_list
for issue in $issue_list; do
    prs_for_issue=`gh api repos/fleetdm/fleet/issues/$issue/timeline --paginate | jq -r '.[]' | $GREP_CMD "fleetdm/fleet/" | $GREP_CMD -oP "pulls\/\K(?:\d+)"`
    echo -n "https://github.com/fleetdm/fleet/issues/$issue"
    if [[ "$prs_for_issue" == "" ]]; then
        echo -n "NO PR's found, please verify they are not missing in the issue, if no PR's were required for this ticket please reconsider adding it to this release."
    fi
    for val in $prs_for_issue; do
        echo -n " $val"
        total_prs+=("$val")
    done
    echo
done


if [ "$force" = "false" ]; then
    read -r -p "Check any issues that have no pull requests, no to cancel and yes to continue? [y/N] " response
    case "$response" in
        [yY][eE][sS]|[yY])
            echo "Continuing to cherry-pick"
            echo
            ;;
        *)
            exit 1
            ;;
    esac
fi

commits=""

for pr in ${total_prs[*]};
do
    output=`gh pr view $pr --json state,mergeCommit,baseRefName`
    state=`echo $output | jq -r .state`
    commit=`echo $output | jq -r .mergeCommit.oid`
    target_branch=`echo $output | jq -r .baseRefName`
    echo -n "$pr $state $commit $target_branch:"
    if [[ "$state" != "MERGED" || "$target_branch" != "main" ]]; then
        echo " WARNING - Skipping pr https://github.com/fleetdm/fleet/pull/$pr"
    else
        if [[ "$commit" != "" && "$commit" != "null" ]]; then
            echo " Commit looks valid - $commit, adding to cherry-pick"
            commits+="$commit "
        else
            echo " WARNING - invalid commit for pr https://github.com/fleetdm/fleet/pull/$pr - $commit"
        fi
    fi
    #echo "======================================="
done

for commit in $commits;
do
    # echo $commit
    timestamp=`git log -n 1 --pretty=format:%at $commit`
    if [ $? -ne 0 ]; then
        echo "Failed to identify $commit, exiting"
        exit 1
    fi
    # echo $timestamp
    time_map[$timestamp]=$commit
done

timestamps=""
failed=false
for key in "${!time_map[@]}"; do
    timestamps+="$key\n"
done
for ts in `echo -e $timestamps | sort`; do
    commit_hash="${time_map[$ts]}"
    # echo "# $ts $commit_hash"
    if git branch --contains "$commit_hash" | $GREP_CMD -q "$(git rev-parse --abbrev-ref HEAD)"; then
        echo "# Commit $commit_hash is on the current branch."
        is_on_current_branch=true
    else
        # echo "# Commit $commit_hash is not on the current branch."
        if [[ "$failed" == "false" ]]; then

            if [ "$dry_run" = "false" ]; then
                git cherry-pick $commit_hash
                if [ $? -ne 0 ]; then
                    echo "Cherry pick of $commit_hash failed. Please resolve then continue the cherry-picks manually"
                    failed=true
                fi
            else
                echo "DRYRUN: Would have cherry picked $commit_hash"
            fi
        else
            echo "git cherry-pick $commit_hash"
        fi
        is_on_current_branch=false
    fi
done

if [[ "$failed" == "false" ]]; then

    if [ "$dry_run" = "false" ]; then
        make changelog
        git diff CHANGELOG.md | $GREP_CMD '^+' | sed 's/^+//g' | $GREP_CMD -v CHANGELOG.md > new_changelog
        prompt=$'I am creating a changelog for an open source project from a list of commit messages. Please format it for me using the following rules:\n1. Correct spelling and punctuation.\n2. Sentence casing.\n3. Past tense.\n4. Each list item is designated with an asterisk.\n5. Output in markdown format.'
        content=$(cat new_changelog | sed -E ':a;N;$!ba;s/\r{0,1}\n/\\n/g')
        question="${prompt}\n\n${content}"

        # API endpoint for ChatGPT
        api_endpoint="https://api.openai.com/v1/chat/completions"
        output="null"

        while [[ "$output" == "null" ]]; do
            data_payload=$(jq -n \
                              --arg prompt "$question" \
                              --arg model "gpt-3.5-turbo" \
                              '{model: $model, messages: [{"role": "user", "content": $prompt}]}')

            response=$(curl -s -X POST $api_endpoint \
               -H "Content-Type: application/json" \
               -H "Authorization: Bearer $open_api_key" \
               --data "$data_payload")

            output=`echo $response | jq -r .choices[0].message.content`
            echo "${output}"
        done
    else
        echo "DRYRUN: Would have run make changelog and sent to ChatGPT to format"
    fi

    if [ "$dry_run" = "false" ]; then
        git checkout CHANGELOG.md
        if [ -n "$target_date" ]; then
            tartget_date=`date +"%b %d, %Y"`
        fi
        echo "## Fleet $target_milestone ($tartget_date)" > temp_changelog
        echo "" >> temp_changelog
        echo "### Bug fixes" >> temp_changelog
        echo "" >> temp_changelog
        echo -e "${output}" >> temp_changelog
        echo "" >> temp_changelog
        cp CHANGELOG.md old_changelog
        cat temp_changelog > CHANGELOG.md
        cat old_changelog >> CHANGELOG.md
        #rm -f temp_changelog
        rm -f old_changelog
        git add CHANGELOG.md
        git commit -m "Adding changes for patch $target_milestone"
        git push origin $target_patch_branch -f

        cp CHANGELOG.md /tmp
        git checkout main 
        git pull origin main
        git checkout -b update-changelog-$target_milestone
        cp /tmp/CHANGELOG.md .
        git add CHANGELOG.md
        git commit -m "Updating changelog for $target_milestone"
        git push origin update-changelog-$target_milestone
        gh pr create -f

        git checkout $target_patch_branch
    else
        echo "DRYRUN: Would have formatted changelog and created PR on main"
    fi

    # Check for QA issue
    if [ "$dry_run" = "false" ]; then
        found=$(gh issue list --search "Release QA: $target_milestone in:title" --json number | jq length)
        if [[ "$found" == "0" ]]; then
            cat .github/ISSUE_TEMPLATE/release-qa.md | awk 'BEGIN {count=0} /^---$/ {count++} count==2 && /^---$/ {getline; count++} count > 2 {print}' > temp_qa_issue_file
            gh issue create --title "Release QA: $target_milestone" -F temp_qa_issue_file \
                --assignee "sabrinabuckets" --assignee "xpkoala" --label ":release" --label "#g-mdm" --label "#g-endpoint-ops"
            rm -f temp_qa_issue_file
        fi
    else
        echo "DRYRUN: Would have searched for and created if not found QA release ticket"
    fi

    if [ "$dry_run" = "false" ]; then
        echo "Waiting for github actions to propogate..."
        show_spinner 200
        # For announce in #help-engineering
        echo
        echo "For announcing in #help-engineering"
        echo "===================================================="
        echo "Release $target_milestone QA ticket and docker publish"
        echo "QA ticket for Release $target_milestone " `gh issue list --search "Release QA: $target_milestone in:title" --json url | jq -r .[0].url`
        echo "Docker Deploy status " `gh run list --workflow goreleaser-snapshot-fleet.yaml --json event,url,headBranch | jq -r "[.[]|select(.headBranch==\"$target_patch_branch\")][0].url"`
        echo "List of tickets pulled into release https://github.com/fleetdm/fleet/milestone/$target_milestone_number"

        echo 
    else
        echo "DRYRUN: Would have printed announce in #help-engineering text w/ qa ticket, deploy to docker link, and milestone issue list link"
    fi

    next_tag="fleet-v$next_ver"
    if [ "$dry_run" = "false" ]; then
        # after passing QA
        #echo "After QA is passed run these commands"
        #echo
        git tag $next_tag
        git push origin $next_tag

        show_spinner 200
    else
        echo "DRYRUN: Would have tagged and pushed $next_tag"
    fi

    if [ "$dry_run" = "false" ]; then
        releaser_out=`gh run list --workflow goreleaser-fleet.yaml --json databaseID,event,headBranch,url | jq "[.[]|select(.headBranch==\"$next_tag\")[0]`
        echo "Releaser running " `echo $releaser_out | jq -r ".url"`

        gh run watch `echo $releaser_out | jq -r ".databaseID"`
    else
        echo "DRYRUN: Would found goreleaser action and waited for it to complete"
    fi


    if [ "$dry_run" = "false" ]; then
        echo "### Changes" > release_notes
        echo "" >> release_notes
        cat temp_changelog >> release_notes
        echo "" >> release_notes
        echo "### Upgrading" >> release_notes
        echo "" >> release_notes
        echo "Please visit our [update guide](https://fleetdm.com/docs/deploying/upgrading-fleet) for upgrade instructions." >> release_notes
        echo "" >> release_notes
        echo "### Documentation" >> release_notes
        echo "" >> release_notes
        echo "Documentation for Fleet is available at [fleetdm.com/docs](https://fleetdm.com/docs)." >> release_notes
        echo "" >> release_notes
        echo "### Binary Checksum" >> release_notes
        echo "" >> release_notes
        echo "**SHA256**" >> release_notes
        echo "" >> release_notes
        echo '\`\`\`' >> release_notes
        gh release download $next_tag -p checksums.txt
        cat checksums.txt >> release_notes
        echo '\`\`\`' >> release_notes

        echo
        echo "============== Release Notes ========================"
        cat release_notes
        echo "============== Release Notes ========================"
    else
        echo "DRYRUN: Would have created release notes text for update."
    fi
    # gh release create -d -t "fleet-v$target_milestone" $target_milestone -n ""
    # gh release create -d -t "fleet-v$target_milestone" $target_milestone -F release_notes
    # rm -f release_notes
else
    # TODO echo what to do
    echo "Placeholder, Cherry pick failed....figure out what to do..."
    exit 1
fi

