
# Fleet events
We sponsor and participate in events so that we can support, connect, engage, and grow the Fleet community. We need to be thoughtful about what events we sponsor or host, and we need to be disciplined in how we run events so we can be efficient and effective.

## How to propose an event:  
It's simple Open an issue: **[Propose an event](https://github.com/fleetdm/confidential/issues/new?template=propose-an-event.md)**



## Event process
There are three phases to running an event at FleetDM,
- **Phase 1:** Propose, review and approve new events
- **Phase 2:** Manage and and execute approved events
- **Phase 3:** Event postgame


#### Phase 1 Propose, review and approve new events
**Objective:**  To ensure that the organization is aligned with the investment of time and resources to execute an event
This process is managed through Fleet Issues and is summarized in a detailed tracking spreadsheet.

See the section "Settle event strategy" below for the process.

##### Settle event strategy (approve proposed events)

Anyone at Fleet can propose a future event. Fleet's [Head of GTM Architecture](https://fleetdm.com/handbook/marketing#team) serves as the project manager for managing the event approval process. Events are settled in advance to provide ample time for strategy and planning. This includes any event that Fleet pays to attend or sponsor. 

The "Settle events strategy" meeting is held on the first Wednesday of every quarter to discuss and lock in all events (conferences, field/sales events, and GitOps workshops) for the next quarter.

The [Content Specialist](https://fleetdm.com/handbook/marketing#team) is the DRI for this meeting. 

Once events have been settled for the upcoming quarter, Fleet does not make changes except in extreme circumstances.

1. Add all upcoming proposed in issues using the template (Propose an event: EVENT_NAME - CITY - YYYY-MM-DD). Approval is tracked and recorded in the ["🫧 Proposed events (not yet settled)" tab](https://docs.google.com/spreadsheets/d/1YQXAX2Q_WnGkAwMYjMbQpV3nbCj7gOBbv7Y0u4twxzQ/edit?gid=1411322737#gid=1411322737) of the 🎪 Events spreadsheet (confidential doc). 
2. Proposed events will include the following information:
  - Event Priority (Scale 1 - 10) where 1 is a top priority
  - Event Name
  - Event Location
  - Event DRI
  - Event Dates
  - Type of Event
  - Theme
  - Event Registration
  - Who from Fleet will attend?
  - Which talk proposal will Fleet submit?
  - Estimated budget, including sponsorship or airfare, and lodging for attendees.
2. Attend the 30m quarterly event strategy meeting with the CMO, Head of GTM Architecture, and Content Specialist.
  - During this meeting, Marketing will decide which events (conferences, field/sales events, and GitOps workshops) Fleet will execute in the **following quarter**.
3. After the meeting, the Content Specialist will communicate the settled events by
  - Moving all settled events to the "All 🎪 Official (planned & settled events)" tab of the 🎪 Events spreadsheet (confidential doc).
  - Using the following template, post a message in the [#oooh-events Slack channel](https://fleetdm.slack.com/archives/C054TGK0H7X).

4. Close all proposed event issues that weren't able to be prioritized with a comment explaining why.

#### Phase 2 Manage and and execute approved events
**Objective** To efficiently plan, organize,track and complete the tasks in order to execute an event. This covers everything from once an event is approved to when the event is finished.  From detail planning to promotion, staff assignments to logistics, events can be complicated projects that need focused attention.

Event execution needs to plan and track the detail decisions supporting:

1. **All Event Logistics.** (Location, Venue, Start and End Date/Time, Event Website, Shipping, Staff schedules, costs, and more)
2. **Speaking sessions:** (Location, time, talk metadata(title, abstract, etc), av requirements, and more)
3. **Event Pre Promotion:** (landing page, blog, social, email, customers, prospects, etc)
4. **Lead capture plans and process:** (scanning process, qualifying questions)
5. **Booth:** Design, messaging, staffing hours and assignments, attire, swag, power, etc.)
6. **Key vendor relationships:** (Event Organizer, booth builder, site logisitcs, scanning tech, av tech)

This will be managed in a structure central document for each event so that attendees and organizers have a central place to find information and collaborate. 

[Planning Doc/Tracking Template](https://docs.google.com/document/d/1Td1XtFClRlOMDuoojXUkJvU8f6MUEjsBacMVRqEJbQQ/edit?tab=t.uych0uenb12p#heading=h.qhf7mkrao68w0

#### Phase 3 event postgame
**Objective** To consistently wrap up an event, gather lessons learned, and ensure the organization follows through with our new relationships.  

After the event there are three important activites that need to be completed.

1. **Update CRM:** The CRM is our single source of truth about our relationships. So, it is critical that all the information from what happened at the event is promptly updated in the CRM.
2. **Follow up:** When we make new friends and connections at an event, we must be prompt in follow through and connecting with them after the event. The CRM is the main way to prompt the right person at Fleet to reach back out and follow up.
3. **Post Mortem:** Learn and improve from the previous event. (gather feedback, review lessons learned, and update processes and strategy)


## **Event execution process**

This page outlines the execution process for Fleet events. It builds upon our general event strategy and goals outlined in the [Fleet events](https://fleetdm.com/handbook/marketing#fleet-events) section of the handbook.

### **Tools and single source of truth**

To keep event planning organized, we separate event information from actionable tasks:

1. **Google Docs (Event overview):** Event status and information are tracked in a single event overview document. This is the single source of truth (SSOT) for the event. It includes key questions and answers, as well as working notes. It does **not** contain tasks.  [Events Working Doc](https://docs.google.com/document/d/1Td1XtFClRlOMDuoojXUkJvU8f6MUEjsBacMVRqEJbQQ/edit?tab=t.40315tbnkz8o#heading=h.tgiaheayil7m)
2. **GitHub issues:** Event tasks are tracked in GitHub. We use parent/child issues for specific tasks to execute an event, tracking the execution from the initial planning stages all the way to completion.

All child tasks in GitHub (e.g., draft and finalize talk title/abstract, design booth, order swag, ship booth kit, promote event) should reference back to the event overview document.

### **GitHub labels**

We use GitHub labels to organize the difference between overall event issues and detailed execution tasks, allowing us to filter and track between overview issues only, and specific events only.  The color coding will help us to visually tell the difference between events.  Note the specific event labels have 6 possible colors defined. These should get re-used, as events are completed.

| Label | Color | Hex code | Definition (when to use it) |
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



### **Event plans**

We utilize two general event plans, which act as templates depending on the scale and type of the event:

1. **Conference:** Used for large conferences and events where we have a booth, speaking slots, lead scanning, and other major logistical needs.  
2. **Workshop/Dinner:** Used for our GitOps workshop series (which often includes a post workshop dinner).
3. **Meetups/Happy Hours** ...wip



### **Partner Involvement at GitOps Workshops**

It is a best practice and a goal for Fleet to have channel partner involvement in every   
GitOps workshop we host. Partner involvement strengthens the workshop's reach, reinforces   
Fleet's channel relationships, and increases qualified attendance.

#### **There are three defined partner engagement types for GitOps workshops:**
1. **Dedicated Partner Workshop:** Fleet hosts the workshop exclusively for a single partner, typically at the partner's office. The audience is the partner's internal staff only — no customers, prospects, or other partners are included. This format is used for partner enablement and training.  
2. **Partner Co-Sponsored Workshop** *(most common):* A single channel partner co-sponsors the workshop alongside Fleet. Only one partner sponsor is permitted per event, and no other partners may attend. Fleet and the co-sponsoring partner each invite from their combined customer and prospect base to drive attendance.  
3. **Fleet-Led Open Workshop:** Fleet runs and owns the workshop independently, with no dedicated local partner. This format is used when there is no established partner presence in the workshop's metro area. Any partner is welcome to attend. To drive registration, the channel manager and account executives run a targeted LinkedIn campaign inviting prospects and customers within the workshop's metro area.

### **Execution process**

Once an event is approved, a Marketing directly responsible individual (DRI) is assigned. From there, the process is divided between the Marketing DRI and the Onsite DRI.

#### **Marketing DRI responsibilities**

* Create the event overview planning doc.  
* Create the parent and child execution issues in GitHub.  
* Assign the execution issues to themselves or the Onsite DRI for parts of the plan. (In many cases, a workshop has been planned locally by the Account Executive; this is where specific issues would be assigned out).  
* Ensure leads are uploaded and properly accounted for in Salesforce (SFDC) post-event.

#### **Onsite DRI responsibilities**

* Manage the details of the facility.  
* Set up and configure the booth, swag, and lead capture tools.  
* Coordinate AV, facilities, and catering.  
* Ensure leads and attendance are actively captured during the event.  
* Pack up the event kit and ship it back.  
* Coordinate with the Marketing DRI to get leads uploaded and processed.

### **Lead capture at events**

Fleet uses one of three methods to capture leads at events, in priority order.

#### **Scenario 1: Official event scanner (preferred)**
If the conference provides a lead scanner or rental device, always use it.
1.  **Pre-event:** Create a dedicated SFDC campaign for the booth (separate from any workshop campaign) before the event starts.
2. **At the event:** Scan badges, flag lead temperature, and export the lead list at the end of each day.
3. **Post-event:** Upload leads to SFDC and share in the event Slack channel. Hot leads should receive outreach within 24 hours.

#### **Scenario 2: Popl (no official scanner)**
When no official scanner is available, use the **Popl** app to scan badges and business cards. Popl is connected to SFDC and leads/scans will be added to the SFDC campaign automatically.

##### **Pre-event setup (marketing DRI)**
1. Create a dedicated Popl campaign matching the SFDC campaign name, with dates covering the full event.  
2. Add all booth staff to the campaign. Send Popl invites to anyone who doesn't have an account yet — they'll see the campaign automatically once set up.  
3. Confirm everyone can see the campaign at least 48 hours before the event.

##### **At the event**
* Confirm the correct campaign is selected before scanning.  
* Add notes to each scan immediately after the conversation.  
* If connectivity is poor or unreliable, revert to [Scenario 3: Google Doc fallback](#scenario-3-google-doc-fallback-no-wi-fi-or-popl-not-working).

**Note:** Popl's badge scan and OCR features require an internet connection to process.

#### **Scenario 3: Google Doc fallback (no Wi-Fi or Popl not working)**
If there is poor service and/or Wi-Fi and Popl is not a viable option, one person from the on-site team should open a new Google Doc and share it with the other team members. Create one tab per staff member inside the doc.

**Guide:** For a step-by-step visual walkthrough of this process, refer to [How to Guide-Capturing leads using Google doc offline](#https://docs.google.com/document/d/1equ2pwOY4Op3m2zTX-316p0vp9b7Du6Zmjjqdgmueoc/edit?usp=sharing).

##### **At the event — instructions for each team member**
1. Open a new tab (one per lead).  
2. Tap **\+** → **Image** → **From camera** to photograph their badge.  
3. After they leave, tap the mic and dictate your note. Always start with time, customer name, lead rating, and your notes:  
     
   `[TIME]: [FULL NAME]: [lead rating — hot / warm / cold]: [your notes]`  
    
4. Once done, open a new tab and repeat for each lead.

##### **Post-event**
When back online, the doc syncs automatically. The event DRI reviews all tabs, transcribes leads into SFDC, and assigns lead temperature based on the notes.

### **Definition of done**

An event's execution is not complete until the **Definition of Done** is met: the Event Overview Doc must be fully updated with post-event outcomes, notes, and final details.

## How to automate workshop creation

The workshop tracking process uses the same GitHub parent/child issue structure as conferences, but with a smaller, workshop-specific set of tasks. Use this script instead of the conference script when running a GitOps workshop.

> If you haven't set up the GitHub CLI yet, install it via [Homebrew](https://brew.sh/) (`brew install gh`), then authenticate with `gh auth login` and `gh auth refresh -s project` before continuing.

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

see new-workshop.sh in the marketing folder



<meta name="maintainedBy" value="akuthiala">
<meta name="title" value="🫧 Marketing event execution">
