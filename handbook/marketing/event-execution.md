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

<meta name="maintainedBy" value="johnjeremiah">
<meta name="title" value="🫧 Marketing Event Execution">
