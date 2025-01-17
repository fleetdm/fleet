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
#  /$$$$$$$$ /$$       /$$$$$$$$ /$$$$$$$$ /$$$$$$$$
# | $$_____/| $$      | $$_____/| $$_____/|__  $$__/
# | $$      | $$      | $$      | $$         | $$
# | $$$$$   | $$      | $$$$$   | $$$$$      | $$
# | $$__/   | $$      | $$__/   | $$__/      | $$
# | $$      | $$      | $$      | $$         | $$
# | $$      | $$$$$$$$| $$$$$$$$| $$$$$$$$   | $$
# |__/      |________/|________/|________/   |__/
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

failed=false

usage() {
    echo "Usage: $0 [options] (optional|start_version)"
    echo ""
    echo "Options:"
    echo "  -a, --and_cherry_pick  This is a minor release and cherry pick. Used for unscheduled minor releases that are patches + a specific feature"
    echo "  -c, --cherry_pick_resolved The script has been run, had merge conflicts, and those have been resolved and all cherry picks completed manually."
    echo "  -d, --dry_run          Perform a trial run with no changes made"
    echo "  -f, --force            Skip all confirmations"
    echo "  -h, --help             Display this help message and exit"
    echo "  -g, --tag              Run the tag step"
    echo "  -m, --minor            Increment to a minor version instead of patch (required if including non-bugs)"
    echo "  -n, --announce_only    Announce the release only, do not publish the release."
    echo "  -o, --open_api_key     Set the Open API key for calling out to ChatGPT"
    echo "  -p, --print            If the release is already drafted then print out the helpful info"
    echo "  -q, --quiet            This will skip notifying in slack"
    echo "  -r, --release_notes    Update the release notes in the named release on github and exit (requires changelog output from running the script previously)."
    echo "  -s, --start_version    Set the target starting version (can also be the first positional arg) for the release, defaults to latest release on github"
    echo "  -t, --target_date      Set the target date for the release, defaults to today if not provided"
    echo "  -u, --publish_release  Set's release from draft to release, deploys to dogfood."
    echo "  -v, --target_version   Set the target version for the release"
    echo ""
    echo "Environment Variables:"
    echo "  OPEN_API_KEY           Open API key used for api requests to chat GPT"
    echo "  SLACK_GENERAL_TOKEN    Slack token to publish via curl to #general"
    echo "  SLACK_HELP_INFRA_TOKEN Slack token to publish via curl to #help-infrastructure"
    echo "  SLACK_HELP_ENG_TOKEN   Slack token to publish via curl to #help-engineering"
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
            echo "Please install latest grep with $(brew install grep)"
            exit 1
        fi
    fi
}

check_gh() {
    gh repo set-default
}

check_required_binaries() {
    local missing_counter=0
    # List of required binaries used in the script
    local required_binaries=("jq" "gh" "git" "curl" "awk" "sed" "make" "ack")

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
    check_gh
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

build_changelog() {
    if [ "$dry_run" = "false" ]; then
        make changelog

        git diff CHANGELOG.md | $GREP_CMD '^+' | sed 's/^+//g' | $GREP_CMD -v CHANGELOG.md > new_changelog
        prompt=$'I am creating a changelog for an open source project from a list of commit messages. Please format it for me using the following rules:\n1. Correct spelling and punctuation.\n2. Sentence casing.\n3. Past tense.\n4. Each list item is designated with an asterisk.\n5. Output in markdown format.'
        if [[ "$minor" == "true" ]]; then
            # Place to make a main targeted prompt
            prompt=$'I am creating a changelog for an open source project from a list of commit messages. Please format it for me using the following rules: Organize updates into three categories: Endpoint Operations, Device Management (MDM), and Vulnerability Management, with all bug fixes and misc. improvements listed under "Bug fixes and improvements". Start each entry with a past tense verb, using hyphens for bullet points. Include specific details for new features, bug fixes, API changes, and any necessary user actions. Note changes in user interfaces, system feedback, and significant architectural updates. Highlight mandatory actions and major impacts, especially for system administrators. Order seemingly important features at the top of their respective lists.'
        fi

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

            output=$(echo $response | jq -r .choices[0].message.content)
            echo "${output}"
        done

        git checkout CHANGELOG.md
        if [[ "$target_date" == "" ]]; then
            tartget_date=$(date +"%b %d, %Y")
        fi
        echo "## Fleet $target_milestone ($tartget_date)" > temp_changelog
        echo "" >> temp_changelog
        echo "### Bug fixes" >> temp_changelog
        echo "" >> temp_changelog
        echo -e "${output}" >> temp_changelog
        echo "" >> temp_changelog
        cp CHANGELOG.md old_changelog
        cat temp_changelog
        echo
        echo "About to write changelog"
        if [ "$force" = "false" ]; then
            read -r -p "Does the above changelog look good (edit temp_changelog now to make changes) (n exits)? [y/N] " response
            case "$response" in
                [yY][eE][sS]|[yY])
                    echo
                    ;;
                *)
                    exit 1
                    ;;
            esac
        fi
        cat temp_changelog > CHANGELOG.md
        cat old_changelog >> CHANGELOG.md
        rm -f old_changelog
        cp CHANGELOG.md /tmp
    else
        echo "DRYRUN: Would have formatted changelog"
    fi
}

changelog_and_versions() {
    branch_for_changelog=$1
    source_branch=$2

    local_exists=$(git branch | $GREP_CMD $branch_for_changelog)
    if [ "$dry_run" = "false" ]; then
        if [[ $local_exists != "" ]]; then
            # Clear previous
            git branch -D $branch_for_changelog
        fi
        git checkout -b $branch_for_changelog
        cp /tmp/CHANGELOG.md .
        git add CHANGELOG.md
        escaped_start_version=$(echo "$start_milestone" | sed 's/\./\\./g')
        version_files=$(ack -l --ignore-dir=tools/release --ignore-dir=articles --ignore-file=is:CHANGELOG.md "$escaped_start_version")
        unameOut="$(uname -s)"
        case "${unameOut}" in
            Linux*)     echo "$version_files" | xargs sed -i "s/$escaped_start_version/$target_milestone/g";;
            Darwin*)    echo "$version_files" | xargs sed -i '' "s/$escaped_start_version/$target_milestone/g";;
            *)          echo "unknown distro to parse version"
        esac
        git add terraform charts infrastructure tools
        git commit -m "Adding changes for Fleet v$target_milestone"
        git push origin $branch_for_changelog -f
        gh pr create -f -B $source_branch
        gh workflow run goreleaser-snapshot-fleet.yaml --ref $source_branch # Manually trigger workflow run
    else
        echo "DRYRUN: Would have created Changelog / verison pr from $branch_for_changelog to $source_branch"
    fi
}

create_qa_issue() {
    if [ "$dry_run" = "false" ]; then
        # Check for QA issue
        found=$(gh issue list --search "Release QA: $target_milestone in:title" --json number | jq length)
        if [[ "$found" == "0" ]]; then
            cat .github/ISSUE_TEMPLATE/release-qa.md | awk 'BEGIN {count=0} /^---$/ {count++} count==2 && /^---$/ {getline; count++} count > 2 {print}' > temp_qa_issue_file
            gh issue create --title "Release QA: $target_milestone" -F temp_qa_issue_file \
                --assignee "pezhub" --assignee "xpkoala" --label ":release" --label "#g-mdm" --label "#g-endpoint-ops"
            rm -f temp_qa_issue_file
        fi
    else
        echo "DRYRUN: Would have searched for and created if not found QA release ticket"
    fi
}

print_announce_info() {
    if [ "$dry_run" = "false" ]; then
        qa_ticket=$(gh issue list --search "Release QA: $target_milestone in:title" --json url | jq -r .[0].url)
        docker_deploy=$(gh run list --workflow goreleaser-snapshot-fleet.yaml --json event,url,headBranch --limit 100 | jq -r "[.[]|select(.headBranch==\"$target_branch\")][0].url")
        echo
        echo "For announcing in #help-engineering"
        echo "===================================================="
        echo "Release $target_milestone QA ticket and docker publish"
        echo "QA ticket for Release $target_milestone " $qa_ticket
        echo "Docker Deploy status " $docker_deploy
        echo "List of tickets pulled into release https://github.com/fleetdm/fleet/milestone/$target_milestone_number"
        echo 
        slack_hook_url=https://hooks.slack.com/services
        app_id=T019PP37ALW
        announce_text="Release $target_milestone QA ticket and docker publish\nQA ticket for Release $target_milestone $qa_ticket\nDocker Deploy status $docker_deploy\nList of tickets pulled into release https://github.com/fleetdm/fleet/milestone/$target_milestone_number"
        if [ "$quiet" = "false" ]; then
            curl -X POST -H 'Content-type: application/json' \
                --data "{\"text\":\"$announce_text\"}" \
                $slack_hook_url/$app_id/$SLACK_HELP_ENG_TOKEN
        fi
    else
        echo "DRYRUN: Would have printed announce in #help-engineering text w/ qa ticket, deploy to docker link, and milestone issue list link"
    fi
}

general_announce_info() {
    if [[ "$minor" == "true" ]]; then
        article_url="https://fleetdm.com/releases/fleet-$target_milestone"
        article_published=$(curl -is "$article_url" | head -n 1 | awk '{print $2}')
        if [[ "$article_published" != "200" ]]; then
            echo "Could't find article at '$article_url'"
            exit 1
        fi

        # TODO Publish Linkedin post about release article here and save url
        linkedin_post_url="https://www.linkedin.com/feed/update/urn:li:activity:7274913563989721088"
    fi
    echo "========================================================================="
    echo "Update osquery Slack Fleet channel topic to say the correct version $next_ver"
    echo "========================================================================="
    # Slack
    slack_hook_url=https://hooks.slack.com/services
    app_id=T019PP37ALW
    announce_text=":cloud: :rocket: The latest version of Fleet is $target_milestone.\nMore info: https://github.com/fleetdm/fleet/releases/tag/$next_tag"
    if [[ "$minor" == "true" ]]; then
        announce_text=":cloud: :rocket: The latest version of Fleet is $target_milestone.\nMore info: https://github.com/fleetdm/fleet/releases/tag/$next_tag\nRelease article: $article_url\nLinkedIn post: $linkedin_post_url"
    fi

    echo -e $announce_text

    if [ "$quiet" = "false" ]; then
        if [ "$dry_run" = "false" ]; then
            curl -X POST -H 'Content-type: application/json' \
                --data "{\"text\":\"$announce_text\"}" \
                $slack_hook_url/$app_id/$SLACK_GENERAL_TOKEN

            curl -X POST -H 'Content    -type: application/json' \
                --data "{\"text\":\"$announce_text\nDogfood Deployed $dogfood_deploy\"}" \
                $slack_hook_url/$app_id/$SLACK_HELP_INFRA_TOKEN
        fi
    fi
}

update_release_notes() {
    if [ "$dry_run" = "false" ]; then
        if [ ! -f temp_changelog ]; then
            echo "cannot find changelog to populate release notes"
            exit 1
        fi
        cat temp_changelog | tail -n +3 > release_notes
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
        echo '```' >> release_notes
        gh release download $next_tag -p checksums.txt --clobber
        cat checksums.txt >> release_notes
        echo '```' >> release_notes

        echo
        echo "============== Release Notes ========================"
        cat release_notes
        echo "============== Release Notes ========================"

        gh release edit --draft -F release_notes $next_tag
    else
        echo "DRYRUN: Would have created release notes based on temp_changelog"
    fi
}

tag() {
    if [ "$dry_run" = "false" ]; then
        current_branch=$(git rev-parse --abbrev-ref HEAD)
        found_version=$(cat CHANGELOG.md | $GREP_CMD $target_milestone)
        if [[ "$found_version" == "" ]]; then
            echo "Can't tag if CHANGELOG pr has not been merged yet"
            exit 1
        fi
        if [[ "$current_branch" != "$target_branch" ]]; then
            echo "Can't tag release if you aren't on '$target_branch'"
            exit 1
        fi

        # Officially tag and push
        git tag $next_tag
        git push origin $next_tag

        # This lets us wait for github actions to trigger
        # we are specifically waiting for goreleaser to start
        # off the `tag` branch ie: fleet-v4.47.2 to watch until it completes
        # The last step of goreleaser is the create the draft release for us to modify later
        show_spinner 200
    else
        echo "DRYRUN: Would have tagged and pushed $next_tag"
    fi

    if [ "$dry_run" = "false" ]; then
        releaser_out=$(gh run list --workflow goreleaser-fleet.yaml --json databaseId,event,headBranch,url | jq "[.[]|select(.headBranch==\"$next_tag\")][0]")
        echo "Releaser running " $(echo $releaser_out | jq -r ".url")

        gh run watch $(echo $releaser_out | jq -r ".databaseId")
    else
        echo "DRYRUN: Would found goreleaser action and waited for it to complete"
    fi

    # Update draft release notes w/ changelog / notes / checksums
    update_release_notes
}


publish() {
    if [ "$dry_run" = "false" ]; then
        if [ "$announce_only" = "false" ]; then
            # TODO more checks to validate we are ready to publish
            gh release edit --draft=false --latest $next_tag
            gh workflow run dogfood-deploy.yml -f DOCKER_IMAGE=fleetdm/fleet:$next_ver
            show_spinner 200
            dogfood_deploy=$(gh run list --workflow=dogfood-deploy.yml --status in_progress -L 1 --json url | jq -r '.[] | .url')
            cd tools/fleetctl-npm && npm publish

            issues=$(gh issue list -m $target_milestone --json number | jq -r '.[] | .number')
            for iss in $issues; do
                is_story=$(gh issue view $iss --json labels | jq -r '.labels | .[] | .name' | grep story)
                # close all non-stories
                if [[ "$is_story" == "" ]]; then
                    echo "Closing #$iss"
                    gh issue close $iss
                fi
            done

            echo "Closing milestone"
            gh api repos/fleetdm/fleet/milestones/$target_milestone_number -f state=closed
        fi
    else
        echo "DRYRUN: Would have published $next_tag / deployed to dogfood / closed non-stories / closed milestone / announced in slack"
    fi

    echo "Send general announce" 
    # Send general announcement in #general
    general_announce_info
}

# Validate we have all commands required to perform this script
check_required_binaries

# Initialize variables for the options
minor_cherry_pick=false
cherry_pick_resolved=false
dry_run=false
force=false
minor=false
announce_only=false
open_api_key=""
start_version=""
target_date=""
target_version=""
print_info=false
publish_release=false
release_notes=false
do_tag=false
quiet=false

# Parse long options manually
for arg in "$@"; do
  shift
  case "$arg" in
    "--and_cherry_pick") set -- "$@" "-a" ;;
    "--cherry_pick_resolved") set -- "$@" "-c" ;;
    "--dry-run") set -- "$@" "-d" ;;
    "--force") set -- "$@" "-f" ;;
    "--help") set -- "$@" "-h" ;;
    "--minor") set -- "$@" "-m" ;;
    "--announce_only") set -- "$@" "-n" ;;
    "--open_api_key") set -- "$@" "-o" ;;
    "--print") set -- "$@" "-p" ;;
    "--quiet") set -- "$@" "-q" ;;
    "--publish_release") set -- "$@" "-u" ;;
    "--release_notes") set -- "$@" "-r" ;;
    "--start_version") set -- "$@" "-s" ;;
    "--tag") set -- "$@" "-g" ;;
    "--target_date") set -- "$@" "-t" ;;
    "--target_version") set -- "$@" "-v" ;;
    *)        set -- "$@" "$arg"
  esac
done

# Extract options and their arguments using getopts
while getopts "acdfhgmno:pqrs:t:uv:" opt; do
    case "$opt" in
        a) minor_cherry_pick=true ;;
        c) cherry_pick_resolved=true ;;
        d) dry_run=true ;;
        f) force=true ;;
        h) usage; exit 0 ;;
        g) do_tag=true ;;
        m) minor=true ;;
        n) announce_only=true ;;
        o) open_api_key=$OPTARG ;;
        p) print_info=true ;;
        q) quiet=true ;;
        r) release_notes=true ;;
        s) start_version=$OPTARG ;;
        t) target_date=$OPTARG ;;
        u) publish_release=true ;;
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

if [ -z "$SLACK_GENERAL_TOKEN" ]; then
    echo "Error: No SLACK_GENERAL_TOKEN environment variable." >&2
    exit 1
fi
if [ -z "$SLACK_HELP_INFRA_TOKEN" ]; then
    echo "Error: No SLACK_HELP_INFRA_TOKEN environment variable." >&2
    exit 1
fi
if [ -z "$SLACK_HELP_ENG_TOKEN" ]; then
    echo "Error: No SLACK_HELP_ENG_TOKEN environment variable." >&2
    exit 1
fi

if [[ "$target_date" != "" ]]; then
    validate_and_format_date $target_date
fi

# ex v4.43.0
if [ -z "$start_version" ]; then
    if [[ "$1" == "" ]]; then
        # grab latest draft excluding test version 9.99.9
        draft=$(gh release list | $GREP_CMD Draft | $GREP_CMD -v 9.99.9)
        if [[ "$draft" != "" ]]; then
            target_version=$(echo $draft | awk '{print $1}' | cut -d '-' -f2)
            start_version=$(gh release list | $GREP_CMD Draft -A1 | tail -n1 | awk '{print $1}' | cut -d '-' -f2)
        else
            start_version=$(gh release list | $GREP_CMD Latest | awk '{print $1}' | cut -d '-' -f2)
        fi
    else
        start_version="$1"
    fi
fi

if [[ $start_version != v* ]]; then
    start_version=$(echo "v$start_version")
fi

if [[ "$target_version" != "" ]]; then
    if [[ $target_version != v* ]]; then
        target_version=$(echo "v$target_version")
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

if [[ "$minor" == "true" ]]; then
    echo "Minor release from $start_version to $next_ver"
    # For scheduled minor releases, we want to branch off of main
    start_ver_tag="main"
else
    echo "Patch release from $start_version to $next_ver"
fi

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
# 4.47.2
start_milestone="${start_version:1}"
# 4.48.0
target_milestone="${next_ver:1}"
# 79
target_milestone_number=$(gh api repos/:owner/:repo/milestones | jq -r ".[] | select(.title==\"$target_milestone\") | .number")
# patch-fleet-v4.48.0
target_branch="rc-patch-fleet-$next_ver"
if [[ "$minor" == "true" ]]; then
    target_branch="rc-minor-fleet-$next_ver"
fi

# fleet-v4.48.0
next_tag="fleet-$next_ver"

if [[ "$target_milestone_number" == "" && "$announce_only" == "false" && $dry_run == false ]]; then
    echo "Missing milestone $target_milestone, Please create one and tie tickets to the milestone to continue"
    exit 1
fi

echo "Found milestone $target_milestone with number $target_milestone_number"

if [ "$print_info" = "true" ]; then
    if [ "$announce_only" = "false" ]; then
        print_announce_info
        exit 0
    fi
fi

if [ "$do_tag" = "true" ]; then
    if [ "$announce_only" = "false" ]; then
        tag
        exit 0
    fi
fi

if [ "$release_notes" = "true" ]; then
    if [ "$announce_only" = "false" ]; then
        update_release_notes
        exit 0
    fi
fi

if [ "$publish_release" = "true" ]; then
    publish
    exit 0
fi


if [ "$cherry_pick_resolved" = "false" ]; then
    # TODO Fail if not found
    if [ "$dry_run" = "false" ]; then
        git fetch
        git checkout $start_ver_tag
        git pull origin $start_ver_tag
    else
        echo "DRYRUN: Would have checked out starting at $start_ver_tag"
    fi

    local_exists=$(git branch | $GREP_CMD $target_branch)

    if [ "$dry_run" = "false" ]; then
        if [[ $local_exists != "" ]]; then
            # Clear previous
            git branch -D $target_branch
        fi
        git checkout -b $target_branch
    else
        echo "DRYRUN: Would have cleared / checked out new branch $target_branch"
    fi

    total_prs=()

    issue_list=$(gh issue list --search 'milestone:"'"$target_milestone"'"' --json number | jq -r '.[] | .number')

    if [[ "$issue_list" == "" && "$dry_run" == "false" ]]; then
        echo "Milestone $target_milestone has no target issues, please tie tickets to the milestone to continue"
        exit 1
    fi

    echo "Issue list for new patch $next_ver"
    echo $issue_list

    for issue in $issue_list; do
        prs_for_issue=$(gh api repos/fleetdm/fleet/issues/$issue/timeline --paginate | jq -r '.[]' | $GREP_CMD "fleetdm/fleet/" | $GREP_CMD -oP "pulls\/\K(?:\d+)")
        echo -n "https://github.com/fleetdm/fleet/issues/$issue"
        if [[ "$prs_for_issue" == "" ]]; then
            echo -n " - No PRs found, please verify they are not missing in the issue."
        fi
        for val in $prs_for_issue; do
            pr_base_ref=$(gh pr view "$val" --json baseRefName | jq -r .baseRefName)
            if [[ "$pr_base_ref" != "main" ]]; then
                echo -n " - PR $val is not based off main. Skipping."
            else
                echo -n " $val"
                total_prs+=("$val")
            fi
        done
        echo
    done

    if [ "$force" = "false" ]; then
        read -r -p "Check any issues that have no pull requests, no to cancel and yes to continue? [y/N] " response
        case "$response" in
            [yY][eE][sS]|[yY])
                echo
                ;;
            *)
                exit 1
                ;;
        esac
    fi

    commits=""

    if [[ "$minor" == "false" || "$minor_cherry_pick" == "true" ]]; then
        echo "Continuing to cherry-pick"
        for pr in ${total_prs[*]};
        do
            output=$(gh pr view $pr --json state,mergeCommit,baseRefName)
            state=$(echo $output | jq -r .state)
            commit=$(echo $output | jq -r .mergeCommit.oid)
            target_branch=$(echo $output | jq -r .baseRefName)
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
            timestamp=$(git log -n 1 --pretty=format:%at $commit)
            if [ $? -ne 0 ]; then
                echo "Failed to identify $commit, exiting"
                exit 1
            fi
            # echo $timestamp
            time_map[$timestamp]=$commit
        done

        timestamps=""
        for key in "${!time_map[@]}"; do
            timestamps+="$key\n"
        done
        for ts in $(echo -e $timestamps | sort); do
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
    fi
fi

if [[ "$failed" == "false" ]]; then
    if [ "$dry_run" = "false" ]; then
        # have to push so we can make the PR's back
        git push origin $target_branch -f
    fi

    build_changelog

    # Create PR for changelog and version to release
    update_changelog_prepare_branch="update-changelog-prepare-$target_milestone"
    changelog_and_versions $update_changelog_prepare_branch $target_branch

    if [ "$dry_run" = "false" ]; then
        # Create PR for changelog and version to main
        git checkout main 
        git pull origin main
    else
        echo "DRYRUN: Would have switched to main and pulled latest"
    fi
    
    update_changelog_branch="update-changelog-$target_milestone"
    changelog_and_versions $update_changelog_branch main

    if [ "$dry_run" = "false" ]; then
        # Back on patch / prepare
        git checkout $target_branch
    else
        echo "DRYRUN: Would have switched back to branch $target_branch"
    fi

    if [[ "$dry_run" = "false" && "$minor" == "false" ]]; then
        # Cherry-pick from update-changelog-branch
        ch_commit=$(git log -n 1 --pretty=format:"%H" $update_changelog_branch)
        git cherry-pick $ch_commit
        git push origin $target_branch -f
    fi

    # Check for QA issue
    create_qa_issue

    if [ "$dry_run" = "false" ]; then
        echo "Waiting for github actions to propogate..."
        show_spinner 200
    fi

    # For announce in #help-engineering
    print_announce_info
else
    # TODO echo what to do
    echo "Placeholder, Cherry pick failed....figure out what to do..."
    exit 1
fi
