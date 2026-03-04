# **Event execution process**

This page outlines the execution process for Fleet events. It builds upon our general event strategy and goals outlined in the [Fleet events](https://fleetdm.com/handbook/marketing#fleet-events) section of the handbook.

## **Tools and single source of truth**

To keep event planning organized, we separate event information from actionable tasks:

1. **Google Docs (Event overview):** Event status and information are tracked in a single event overview document. This is the single source of truth (SSOT) for the event. It includes key questions and answers, as well as working notes. It does **not** contain tasks.  [Events Working Doc](https://docs.google.com/document/d/1Td1XtFClRlOMDuoojXUkJvU8f6MUEjsBacMVRqEJbQQ/edit?tab=t.40315tbnkz8o#heading=h.tgiaheayil7m)
2. **GitHub issues:** Event tasks are tracked in GitHub. We use parent/child issues for specific tasks to execute an event, tracking the execution from the initial planning stages all the way to completion.

All child tasks in GitHub (e.g., draft and finalize talk title/abstract, design booth, order swag, ship booth kit, promote event) should reference back to the event overview document.

## **GitHub labels**

We use GitHub labels to organize the difference between overall event issues and detailed execution tasks, allowing us to filter and track between overview issues only, and specific events only.  The color coding will help us to visually tell the difference between events.  Note the specific event labels have 6 possible colors defined. These should get re-used, as events are completed.

| Label | Color | Hex Code | Definition (When to use it) |
| :---- | :---- | :---- | :---- |
| **:mktg-event** | Orange | \#F97316 | The standard label for all events. |
| **:mktg-event:tp** | Dark Rust | \#9A3412 | Indicates this issue is part of event execution in general. |
| **:mktg-event:overview** | Light Peach | \#FFDED2 | The parent issue for the event. |
| **:mktg-event:detail** | Amber | \#F59E0B | Used for detailed tasks (children) of the overall event. |
| **:mktg-event:YYMM-eventname-city** | Sunset Red | \#EF4444 | A first custom label created for each specific event to group a family of tasks together. |
| **:mktg-event:YYMM-eventname-city** | Tangerine | \#FF8A65 | 2nd color custom lable for specific events |
| **:mktg-event:YYMM-eventname-city** | Marigold | \#FBBF24 | 3rd color custom lable for specific events |
| **:mktg-event:YYMM-eventname-city** | Terracotta | \#C2410C | 4th color custom lable for specific events |
| **:mktg-event:YYMM-eventname-city** | Salmon | \#FA8072 | 5th color custom lable for specific events |
| **:mktg-event:YYMM-eventname-city** | Brick | \#B91C1C | 6th color custom lable for specific events |

## **Event plans**

We utilize two general event plans, which act as templates depending on the scale and type of the event:

1. **Conference:** Used for large conferences and events where we have a booth, speaking slots, lead scanning, and other major logistical needs.  
2. **Workshop/Happy hour:** Used for our GitOps workshop series (which often includes happy hours). This smaller template can be used for the full workshop or for bespoke, standalone happy hours.

## **Execution process**

Once an event is approved, a Marketing directly responsible individual (DRI) is assigned. From there, the process is divided between the Marketing DRI and the Onsite DRI.

### **Marketing DRI responsibilities**

* Create the event overview planning doc.  
* Create the parent and child execution issues in GitHub.  
* Assign the execution issues to themselves or the Onsite DRI for parts of the plan. (In many cases, a workshop has been planned locally by the Account Executive; this is where specific issues would be assigned out).  
* Ensure leads are uploaded and properly accounted for in Salesforce (SFDC) post-event.

### **Onsite DRI responsibilities**

* Manage the details of the facility.  
* Set up and configure the booth, swag, and lead capture tools.  
* Coordinate AV, facilities, and catering.  
* Ensure leads and attendance are actively captured during the event.  
* Pack up the event kit and ship it back.  
* Coordinate with the Marketing DRI to get leads uploaded and processed.

## **Definition of done**

An event's execution is not complete until the **Definition of Done** is met: the Event Overview Doc must be fully updated with post-event outcomes, notes, and final details.

## **How to automate event creation**

Since the tracking process uses github issues and subissues to track tasks, it can be tedious to create the structure for a new event. Here are the steps to automate the creation of the event execution issues in github.

### Setup
We will use a local script that executes commands on the local GitHub command line interface (CLI).  In order to get started you need to have the GitHub CLI installed.

1. **Install Homebrew.**  *Homebrew is a package installer and the simplest way to get the GitHub CLI installed.*.  
    1. Navigate to https://brew.sh/ and copy the installation script.
	2. Open a terminal window, paste and run the script.  *You will be prompted for your local password*
2. **Install GitHub CLI (GH CLI)**
    Using homebrew, tell it to install the GitHub command line interface
	`brew install gh`
	          
3. **Authenticate / connect with GitHub**
   1. Enter the command: `gh auth login` and follow the steps to authenticate with the repository

4. **Update the CLI permissions to include projects**
   1. Enter the command `gh auth refresh -s project` and follow the steps to authenticate.

TODO - add a test set up section where a user has a simple issue script they test


### Event Template Process and Script
Creating a new event group is now simple.
1. copy the script below and save as **NewEvent.sh**
2. Edit the script.

First - CHANGE THESE THREE THINGS.   

**Nothing else needs to change**

* EVENT_SLUG - will be the name of the event and part of the label for the event
* PLANNING_DOC_URL - is the link to the google doc where we're keeping the latest status of the event plans
* REQUEST_ISSUE - is thte number of the issue that proposed the event

For example:
```
EVENT_SLUG="2606-MacDevOpsYVR-Montreal"
PLANNING_DOC_URL="https://docs.google.com/document/d/1Td1XtFClRlOMDuoojXUkJvU8f6MUEjsBacMVRqEJbQQ/edit?tab=t.afz38t4pwdka"
REQUEST_ISSUE = "#14599"   
```   

Save the changed file **NewEvent.sh**

And then execute the script. `./NewEvent.sh`

That's it.  This should create the events in GitHub to manage the event.

here's the script

```bash
#!/bin/bash
# --- Event specifics / details - change this
EVENT_SLUG="2606-MacDevOpsYVR-Montreal"
PLANNING_DOC_URL="https://docs.google.com/document/d/1Td1XtFClRlOMDuoojXUkJvU8f6MUEjsBacMVRqEJbQQ/edit?tab=t.afz38t4pwdka"
REQUEST_ISSUE = "#14599"

# No need to change anything else to run the script

# --- Static Configuration ---
ORG="fleetdm"
REPO="confidential"
PROJECT_NUMBER="94"

# 1. Define Labels (Fixed with leading colons)
NEW_LABEL=":mktg-event:${EVENT_SLUG}"
PARENT_LABELS=":mktg-event,:mktg-event:overview,:mktg-event:tp"
CHILD_LABELS=":mktg-event,:mktg-event:detail,:mktg-event:tp"

# Ensure the specific event label exists
echo "1. Ensuring label '${NEW_LABEL}' exists..."
# gh label create "$NEW_LABEL" --repo "$ORG/$REPO" --color "1D76DB" --force >/dev/null 2>&1 || true
gh label create "$NEW_LABEL" --repo "$ORG/$REPO" --force >/dev/null 2>&1 || true


# ==========================================
# STEP 1: CREATE THE PARENT ISSUE
# ==========================================
echo "2. Creating Parent Issue (Overview)..."

PARENT_TITLE="${EVENT_SLUG} Execution Overview"

# Using a Heredoc for clean, WYSIWYG Markdown formatting
PARENT_BODY=$(cat << EOF
Master tracking issue for ${EVENT_SLUG}.

EXECUTION for request $REQUEST_ISSUE

## Executive Snapshot & Key Decisions 

Use this section for a quick overview. If someone only reads this part, they should understand the scope and scale of our presence. 

| Category | Details |
|---|---|
| Event Name | [e.g., KubeCon NA 2026] |
| Dates | [Start Date] to [End Date] |
| Location | [City, State, Venue Name] |
| Event Website | [Link to official site] |
| Budget Estimate | [Total estimated cost] |
| Primary Goal | [e.g., Lead Gen (500 scans), Brand Awareness, Recruiting] |
| Booth Size | [e.g., 10x20, Island, Tabletop] |
| Speaking Slot? (details below) | Yes or No |
| Workshop? | Yes or No |
| DRI | [Name of person responsible] |
| Onsite DRI | [Name of person responsible] |
| Planing Doc | $PLANNING_DOC_URL |

- [ ] Finalize sponsorship agreements
- [ ] Assign issues/tasks

## Progress Tracker
- [ ] 1. Speaking Session & Workshop Details
- [ ] 2. Promotion & Marketing Plan
- [ ] 3. Booth Strategy & Messaging
- [ ] 4. Staffing & Travel Logistics
- [ ] 5. Execution, Logistics & Swag
- [ ] 6 Lead Capture & Follow-Up Strategy
- [ ] 7. Post-Mortem & ROI Analysis
EOF
)

# Create Parent
PARENT_URL=$(gh issue create \
  --repo "$ORG/$REPO" \
  --title "$PARENT_TITLE" \
  --body "$PARENT_BODY" \
  --label "$PARENT_LABELS,$NEW_LABEL")

# CRITICAL CHECK: Did the parent issue actually get created?
if [ -z "$PARENT_URL" ]; then
    echo "❌ ERROR: Failed to create Parent Issue. Check if labels exist in the repo."
    exit 1
fi

# Extract Issue Number from URL
PARENT_NUM=$(echo "$PARENT_URL" | awk -F/ '{print $NF}')
echo "   ✅ Parent Created: #$PARENT_NUM"

# Get the Global Node ID of the Parent (Needed for GraphQL linking)
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

# Add Parent to Project
gh project item-add "$PROJECT_NUMBER" --owner "$ORG" --url "$PARENT_URL" >/dev/null 2>&1


# ==========================================
# STEP 2: HELPER FUNCTION FOR CHILD ISSUES
# ==========================================
# This function handles the creation and linking of all sub-issues
create_sub_issue() {
    local TITLE="$1"
    local BODY="$2"
    
    CHILD_URL=$(gh issue create \
      --repo "$ORG/$REPO" \
      --title "$TITLE: ${EVENT_SLUG}" \
      --body "$BODY" \
      --label "$CHILD_LABELS,$NEW_LABEL")
      
    if [ -n "$CHILD_URL" ]; then
        CHILD_NUM=$(echo "$CHILD_URL" | awk -F/ '{print $NF}')
        
        # Get Child Node ID
        CHILD_NODE_ID=$(gh api graphql -f query='
          query($owner:String!, $repo:String!, $number:Int!) { 
            repository(owner:$owner, name:$repo) { 
              issue(number:$number) { id } 
            } 
          }' -f owner="$ORG" -f repo="$REPO" -F number="$CHILD_NUM" --jq '.data.repository.issue.id')

        echo "   ✅ Created Child: #$CHILD_NUM ($TITLE)"
        
        # Link as Sub-Issue via GraphQL
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

        # Add to Project
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
BODY=$(cat << EOF
**Description**
Track all details, deadlines, and requirements for any speaking slots or workshops we are hosting before, during, or after the event.

- [ ] Confirm speaking session details (Title, Speaker, Date/Time, Room)
- [ ] Submit Abstract Link and AV Requirements
- [ ] Confirm workshop hosting and timing
- [ ] Update Workshop Planning Doc, Registration Link, and Capacity
- [ ] Update the $PLANNING_DOC_URL
EOF
)
create_sub_issue "1. Speaking Session & Workshop Details" "$BODY"


# --- Child 2 ---
BODY=$(cat << EOF
**Description**
Manage how we are driving traffic to our booth, session, or workshop.

- [ ] Schedule Pre-Event Email Blast 
- [ ] Schedule LinkedIn and Twitter/X Posts
- [ ] Create Speaker Promo Graphics and Blog Post 
- [ ] Assign Customer Invites
- [ ] Assign Live Social Coverage during event 
- [ ] Schedule Event App Push Notification
- [ ] Update the $PLANNING_DOC_URL
EOF
)
create_sub_issue "2. Promotion & Marketing Plan" "$BODY"


# --- Child 3 ---
BODY=$(cat << EOF
**Description**
Define the core purpose, layout, and messaging for our physical footprint on the show floor.

- [ ] Document Booth Number and Exhibit Hall Hours 
- [ ] Define Core Messaging/Theme 
- [ ] List Key Demos 
- [ ] Document Key Requirements (internet, scanners, monitors)
- [ ] Update the $PLANNING_DOC_URL
EOF
)
create_sub_issue "3. Booth Strategy & Messaging" "$BODY"


# --- Child 4 ---
BODY=$(cat << EOF
**Description**
Manage who is going, where they are staying, and when they are working the booth.

- [ ] Assign Staff Manager and Attire 
- [ ] Select Suggested Hotel 
- [ ] Set Arrival and Departure Requirements 
- [ ] Complete Staff Assignments table 
- [ ] Create Booth Staffing Schedule
- [ ] Update the $PLANNING_DOC_URL
EOF
)
create_sub_issue "4. Staffing & Travel Logistics" "$BODY"


# --- Child 5 ---
BODY=$(cat << EOF
**Description**
This section is for the operations team to handle on-site setup, booth build, and shipping.

- [ ] Track Shipping & Handling deadlines and tracking numbers 
- [ ] Create Return Shipping Label 
- [ ] Confirm Booth Vendor, Graphics Deadline, and Furniture/Electrical 
- [ ] Order Premium Swag, General Swag, and Raffle/Contest items 
- [ ] Complete Key Points of Contact table
- [ ] Update the $PLANNING_DOC_URL
EOF
)
create_sub_issue "5. Execution, Logistics & Swag" "$BODY"


# --- Child 6 ---
BODY=$(cat << EOF
**Description**
Crucial for ROI. Track how we capture data and what happens next.

- [ ] Define Capture Mechanics, Method, and Device Rental 
- [ ] Define Incentive to Scan 
- [ ] Write Qualifying Questions for Booth Staff 
- [ ] Assign Lead Upload Owner and SLA 
- [ ] Define Follow Up Strategy and Nurture Sequence
- [ ] Update the $PLANNING_DOC_URL
EOF
)
create_sub_issue "6. Lead Capture & Follow-Up Strategy" "$BODY"


# --- Child 7 ---
BODY=$(cat << EOF
**Description**
To be filled out within 1 week of event conclusion to analyze performance and ROI.

- [ ] Record The Numbers (Leads, MQLs, Spend, CPL) 
- [ ] Complete Retrospective (What went well/wrong) 
- [ ] Document Competitor Intel 
- [ ] Upload Photo Archive
- [ ] Update the $PLANNING_DOC_URL
EOF
)
create_sub_issue "7. Post-Mortem & ROI Analysis" "$BODY"

echo "Done."
```


## How to automate workshop creation

The workshop tracking process uses the same GitHub parent/child issue structure as conferences, but with a smaller, workshop-specific set of tasks. Use this script instead of the conference script when running a GitOps workshop (with or without a happy hour).

> If you haven't set up the GitHub CLI yet, follow the **Setup** steps in the [How to automate event creation](#how-to-automate-event-creation) section above before continuing.

### Workshop template process and script

Creating a new workshop issue group is straightforward.

1. Copy the script below and save it as **NewWorkshop.sh**
2. Edit the script.

First — **CHANGE THESE THREE THINGS.**

**Nothing else needs to change.**

- `WORKSHOP_SLUG` — will be the name of the workshop and part of the GitHub label
- `PLANNING_DOC_URL` — link to the Google Doc where the latest workshop status is tracked
- `REQUEST_ISSUE` — the number of the issue that proposed the workshop

For example:

```
WORKSHOP_SLUG="2606-GitOps-Workshop-Montreal"
PLANNING_DOC_URL="https://docs.google.com/document/d/YOUR_PLANNING_DOC_ID/edit"
REQUEST_ISSUE="#00000"
```

Save the changed file **NewWorkshop.sh**, then run it:

```
./NewWorkshop.sh
```

This will create a parent overview issue and six linked child issues in GitHub to manage the full workshop lifecycle.

Here's the script:

```bash
#!/bin/bash
# --- Workshop specifics / details - change this
WORKSHOP_SLUG="2606-GitOps-Workshop-Montreal"
PLANNING_DOC_URL="https://docs.google.com/document/d/YOUR_PLANNING_DOC_ID/edit"
REQUEST_ISSUE="#00000"

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

Use this section for a quick overview. If someone only reads this part, they should understand the scope and scale of our workshop.

| Category | Details |
|---|---|
| Workshop Name | [e.g., GitOps Workshop — Atlanta] |
| Date | [Date] |
| Location | [City, Venue Name] |
| Capacity | [Max Attendees] |
| Lead Instructor | @[Name] |
| Onsite DRI | @[Name] |
| Marketing DRI | @[Name] |
| Happy Hour? | Yes or No |
| Planning Doc | $PLANNING_DOC_URL |

- [ ] Confirm workshop date and venue
- [ ] Assign issues/tasks to DRIs

## Progress Tracker
- [ ] 1. Workshop Promotion & Registration Launch
- [ ] 2. Venue Selection & Logistics
- [ ] 3. Happy Hour Planning & Promotion
- [ ] 4. Workshop Catering
- [ ] 5. Travel & Staffing
- [ ] 6. Post-Mortem & Follow-Up
EOF

PARENT_URL=$(gh issue create \
  --repo "$ORG/$REPO" \
  --title "$PARENT_TITLE" \
  --body-file "$BODY_FILE" \
  --label "$PARENT_LABELS,$NEW_LABEL")

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
    # Body is already written to $BODY_FILE by the caller

    CHILD_URL=$(gh issue create \
      --repo "$ORG/$REPO" \
      --title "$TITLE: ${WORKSHOP_SLUG}" \
      --body-file "$BODY_FILE" \
      --label "$CHILD_LABELS,$NEW_LABEL")

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

Note: You can launch with "Venue TBD" or "Downtown [City]" if the specific room isn't booked yet.

**Who:** Marketing DRI

- [ ] Create Workshop Landing Page on Eventbrite or Luma
- [ ] Schedule Email Blast to target audience
- [ ] Schedule LinkedIn and Twitter/X posts and request speaker graphics
- [ ] Notify AEs and Partners to drive personal invites
- [ ] Monitor registration — watch for waitlists or low attendance and adjust promo if needed
- [ ] Update the $PLANNING_DOC_URL with "Promotion & Marketing Plan" details and registration link
EOF
create_sub_issue "1. Workshop Promotion & Registration Launch"


# --- Child 2 ---
cat > "$BODY_FILE" << EOF
**Description**
Secure the physical space for the workshop. Once confirmed, notify attendees.

**Who:** Onsite DRI

- [ ] Secure venue — confirm availability for workshop date
- [ ] Power audit — confirm every seat has access to power, or plan to bring extension cords
- [ ] AV check — confirm projector/HDMI availability
- [ ] Update Workshop Landing Page with the specific venue name and address
- [ ] Update the $PLANNING_DOC_URL with "Venue Details" section
EOF
create_sub_issue "2. Venue Selection & Logistics"


# --- Child 3 ---
cat > "$BODY_FILE" << EOF
**Description**
Plan the post-workshop networking. This is treated as a separate event to allow for broader networking — invite people who couldn't make the workshop itself.

**Who:** Onsite DRI

- [ ] Secure venue — find a bar/restaurant within a 5-minute walk of the workshop area
- [ ] Confirm menu/tab — decide on Open Bar vs. Fixed Menu and set the budget cap
- [ ] Create Happy Hour registration page on Eventbrite or Luma and promote separately
- [ ] Schedule LinkedIn/Twitter posts promoting the Happy Hour
- [ ] Update the $PLANNING_DOC_URL with "Post-Event Happy Hour" section including venue and registration link
EOF
create_sub_issue "3. Happy Hour Planning & Promotion"


# --- Child 4 ---
cat > "$BODY_FILE" << EOF
**Description**
Finalize in-room food and drink orders.

Wait to complete this until ~1 week before the event so you have an accurate headcount.

**Who:** Onsite DRI

- [ ] Check registration count — confirm headcount from Registered and Waitlist to avoid over-ordering
- [ ] Order food and drinks
- [ ] Update the $PLANNING_DOC_URL with "Catering" section and order details
EOF
create_sub_issue "4. Workshop Catering"


# --- Child 5 ---
cat > "$BODY_FILE" << EOF
**Description**
Ensure the instructor and support staff can get to the city and are prepared for the event.

**Who:** Onsite DRI, Marketing DRI, Attendees

- [ ] Book flights for Lead Instructor and TA if traveling
- [ ] Book hotel — ensure proximity to the venue
- [ ] Confirm staffing assignments and attire
- [ ] Update the $PLANNING_DOC_URL with "Staff Travel" and "Logistics" sections
EOF
create_sub_issue "5. Travel & Staffing"


# --- Child 6 ---
cat > "$BODY_FILE" << EOF
**Description**
To be completed within 48 hours after the event. Close the loop on leads and technical feedback.

- [ ] Calculate stats — record Registered, Attended, and No-Show rates for both Workshop and Happy Hour
- [ ] Log technical issues — document any WiFi drops or firewall blockers for future reference
- [ ] CRM upload — upload attendee list to Salesforce/HubSpot
- [ ] Send follow-up email with slides and repo links
- [ ] Update the $PLANNING_DOC_URL with completed "Post-Mortem & Follow-Up" section
EOF
create_sub_issue "6. Post-Mortem & Follow-Up"

echo "Done."
```


<meta name="maintainedBy" value="johnjeremiah">
<meta name="title" value="🫧 Marketing Event Execution">
