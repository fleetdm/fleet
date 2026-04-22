#!/bin/bash
# --- Workshop specifics / details - change this
WORKSHOP_SLUG="2026_05-GitOps-Workshop-test3"
PLANNING_DOC_URL="https://docs.google.com"
REQUEST_ISSUE="#14944"
LOCATION="test3"
MKTG_DRI="mb-chigoose312"
CHANNEL_DRI="escomeau"
SOCIAL_DRI="tombasgil"
DESIGN_DRI="mike-j-thomas"

# No need to change anything else to run the script

# --- Static Configuration ---
ORG="fleetdm"
REPO="confidential"
PROJECT_NUMBER="94"

# 1. Define Labels
NEW_LABEL=":mktg-event:${WORKSHOP_SLUG}"
PARENT_LABELS=":mktg-event,:mktg-event:overview,:mktg-event:tp"
CHILD_LABELS=":mktg-event,:mktg-event:detail,:mktg-event:tp"

# Temp file for issue bodies (avoids heredoc/subshell parenthesis conflicts)
BODY_FILE=$(mktemp)
cleanup() { rm -f "$BODY_FILE"; }
trap cleanup EXIT

# Ensure the specific workshop label exists
echo "1. Ensuring label '${NEW_LABEL}' exists..."
gh label create "$NEW_LABEL" --repo "$ORG/$REPO" --force >/dev/null 2>&1 || true


# ==========================================
# STEP 1: CREATE THE PARENT ISSUE
# ==========================================
echo "2. Creating Parent Issue (Overview)..."

PARENT_TITLE="${WORKSHOP_SLUG} Workshop Overview"

cat > "$BODY_FILE" << EOF
Master tracking issue for the GitOps Workshop: ${WORKSHOP_SLUG}.

EXECUTION for request $REQUEST_ISSUE

## Executive Snapshot & Key Decisions
- [ ] Update the working google doc for the ${EVENT_SLUG}
See: Planing Doc: $PLANNING_DOC_URL 

- [ ] Finalize sponsorship agreements
- [ ] Assign child issues/tasks

## Progress Tracker - See SubIssues

EOF

PARENT_URL=$(gh issue create \
  --repo "$ORG/$REPO" \
  --title "$PARENT_TITLE" \
  --body-file "$BODY_FILE" \
  --label "$PARENT_LABELS,$NEW_LABEL" \
  --assignee "$MKTG_DRI") 

if [ -z "$PARENT_URL" ]; then
    echo "❌ ERROR: Failed to create Parent Issue. Check if labels exist in the repo."
    exit 1
fi

PARENT_NUM=$(echo "$PARENT_URL" | awk -F/ '{print $NF}')
echo "   ✅ Parent Created: #$PARENT_NUM"

PARENT_NODE_ID=$(gh api graphql -f query='
  query($owner:String!, $repo:String!, $number:Int!) {
    repository(owner:$owner, name:$repo) {
      issue(number:$number) { id }
    }
  }' -f owner="$ORG" -f repo="$REPO" -F number="$PARENT_NUM" --jq '.data.repository.issue.id')

if [ -z "$PARENT_NODE_ID" ] || [ "$PARENT_NODE_ID" == "null" ]; then
    echo "❌ ERROR: Could not fetch GraphQL Node ID for Parent #$PARENT_NUM"
    exit 1
fi

echo "   🔹 Parent Node ID: $PARENT_NODE_ID"
gh project item-add "$PROJECT_NUMBER" --owner "$ORG" --url "$PARENT_URL" >/dev/null 2>&1


# ==========================================
# STEP 2: HELPER FUNCTION FOR CHILD ISSUES
# ==========================================
create_sub_issue() {
    local TITLE="$1"
    local ASSIGNEE="$2"
    # Body is already written to $BODY_FILE by the caller

    local OPTIONAL_ARGS=()
    if [ -n "$ASSIGNEE" ]; then
        OPTIONAL_ARGS=(--assignee "$ASSIGNEE")
    fi

    CHILD_URL=$(gh issue create \
      --repo "$ORG/$REPO" \
      --title "$TITLE: ${WORKSHOP_SLUG}" \
      --body-file "$BODY_FILE" \
      --label "$CHILD_LABELS,$NEW_LABEL" \
      "${OPTIONAL_ARGS[@]}")

    if [ -n "$CHILD_URL" ]; then
        CHILD_NUM=$(echo "$CHILD_URL" | awk -F/ '{print $NF}')

        CHILD_NODE_ID=$(gh api graphql -f query='
          query($owner:String!, $repo:String!, $number:Int!) {
            repository(owner:$owner, name:$repo) {
              issue(number:$number) { id }
            }
          }' -f owner="$ORG" -f repo="$REPO" -F number="$CHILD_NUM" --jq '.data.repository.issue.id')

        echo "   ✅ Created Child: #$CHILD_NUM ($TITLE)"

        LINK_RESULT=$(gh api graphql -f query='
          mutation($parentId: ID!, $childId: ID!) {
            addSubIssue(input: {issueId: $parentId, subIssueId: $childId}) {
              clientMutationId
            }
          }
        ' -f parentId="$PARENT_NODE_ID" -f childId="$CHILD_NODE_ID" 2>&1)

        if [[ $? -eq 0 ]]; then
            echo "      🔗 Linked as Sub-issue to Parent #$PARENT_NUM"
        else
            echo "      ⚠️ Failed to link. Error details:"
            echo "$LINK_RESULT"
        fi

        gh project item-add "$PROJECT_NUMBER" --owner "$ORG" --url "$CHILD_URL" >/dev/null 2>&1
    else
        echo "   ❌ Failed to create child: $TITLE"
    fi
}


# ==========================================
# STEP 3: DEFINE & CREATE CHILD ISSUES
# ==========================================
echo "3. Creating and Linking Child Issues..."

# --- Child 1 ---
cat > "$BODY_FILE" << EOF
**Description**
Get the main workshop event live to start gathering leads.

Note: You can launch with "Venue TBD" or "Downtown [City]" if the specific room isn't booked yet- SEE TEMPLATE LINK BELOW

**Who:** Marketing DRI

- [ ] Update the $PLANNING_DOC_URL with these details
- Create Workshop Landing Page on Eventbrite or Luma
Link to [TEMPLATE in EventBrite](https://www.eventbrite.com/e/1985443830945?aff=oddtdtcreator)
- Create Slack Channel
- Create Calendar Placeholder
- Create SFDC campaign
- Identify Staff and Instructor(s)
EOF
create_sub_issue "1. Create & Launch Workshop" "$MKTG_DRI"

# --- Child 2 ---
cat > "$BODY_FILE" << EOF
**Description**
Ensure the instructor and support staff can get to the city and are prepared for the event.

**Who:** Onsite DRI, Marketing DRI, Attendees
- [ ] Update the $PLANNING_DOC_URL with these details
- Book flights for Lead Instructor and TA if traveling
- Book hotel — ensure proximity to the venue
- Confirm staffing assignments and attire
EOF
create_sub_issue "2. Travel & Staffing" "$MKTG_DRI"

# --- Child 3 ---
cat > "$BODY_FILE" << EOF
**Description**
Secure the physical space for the workshop. Once confirmed, notify attendees.

**Who:** Onsite DRI, Marketing DRI
- [ ] Update the $PLANNING_DOC_URL with these details
- Secure venue — confirm availability for workshop date
- Power + AV check — confirm power drops options + projector/HDMI availability
- F&B: Identify options - ask venue POC for F&B options. 
- F&B: Decide amount and place order (example: assortment of drinks and light snacks)
- Update Workshop Landing Page with the specific venue name and address
EOF
create_sub_issue "3. Venue Selection + Food & Beverage" "$MKTG_DRI"

# --- Child 4 ---
cat > "$BODY_FILE" << EOF
**Description**
Each workshop requires looping in our go-to local partner and assigning the channel manager to the planning team to lead coordination efforts.. 

**Who:** Marketing DRI
- [ ] Update the $PLANNING_DOC_URL with these details
- Assign issue to Eric Comeau
- Connect with AE to discuss which partner to invite/include on the workshop.
- Share Workshop reg link with identified partner to help drive registration
EOF
create_sub_issue "4. Engage the Channel" "$CHANNEL_DRI"

# --- Child 5 ---
cat > "$BODY_FILE" << EOF
**Description**
For each event and workshop we do, we want to have a special sticker created, tailored to that city.   

**Who:** Design DRI
- [ ] Update the $PLANNING_DOC_URL with these details
- Assign issue to Mike Thomas
- Create a Fleet sticker graphic customized to the city where workshop is being held.  
- Send Sticker graphic to the Marketing DRI 7 days before the event date.  
EOF
create_sub_issue "5. Design" "$DESIGN_DRI"

# --- Child 6 ---
cat > "$BODY_FILE" << EOF
**Description**
General + Instructor social posts for team use to help drive awareness and registration. 

**Who:** Marketing DRI
- [ ] Update the $PLANNING_DOC_URL with these details
- Assign issue to Tom Basgil
- Create LinkedIn + Twitter/X posts for the following: General Workshop itself plus an Instructor post
- Share the posts with the attending team & instructor on the event slack channel (link to slack channel located in the event doc) 
EOF
create_sub_issue "6. Get Social" "$SOCIAL_DRI"


# --- Child 7 ---
cat > "$BODY_FILE" << EOF
**Description**
Plan the post-workshop networking. This is treated as a separate event to allow for broader networking — invite people who couldn't make the workshop itself.

**Who:** Onsite DRI, Marketing DRI
- [ ] Update the $PLANNING_DOC_URL with these details
- Secure venue — find a bar/restaurant within a 5-minute walk of the workshop area
- Confirm menu/tab — decide on Open Bar vs. Fixed Menu and set the budget cap
- IF NEEDED: Create Dinner registration page on Eventbrite or Luma and promote separately
EOF
create_sub_issue "7. Dinner Planning" "$MKTG_DRI"


# --- Child 8 ---
cat > "$BODY_FILE" << EOF
**Description**
To be completed within 48 hours after the event. Close the loop on leads and technical feedback.
- [ ] Update the $PLANNING_DOC_URL with these details
- Calculate stats — record Registered, Attended, and No-Show rates for both Workshop and Dinner
- Log technical issues — document any WiFi drops or firewall blockers for future reference
- CRM upload — upload attendee list to Salesforce/HubSpot
- Send follow-up email with slides and repo links

EOF
create_sub_issue "8. Post-Mortem & Follow-Up" "$MKTG_DRI"

echo "Done."