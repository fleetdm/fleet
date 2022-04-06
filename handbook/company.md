# Company

## About Fleet

Fleet Device Management Inc is an [open core company](https://www.heavybit.com/library/video/commercial-open-source-business-strategies/) that sells subscriptions that offer [more features and support](https://fleetdm.com/pricing) for Fleet and [osquery](https://osquery.io), the leading open source endpoint agent.

We are dedicated to:

- 🧑‍🚀 automating IT and security
- 💍 reducing the proliferation of agents and growing the adoption of osquery (one agent to rule them all)
- 🪟 privacy, transparency, and trust through open source software
- 👁️ remaining the freshest, simplest source of truth for every kind of computing device and OS
- 💻 building a better way to manage computers


## Culture

### All remote
Fleet Device Management Inc. is an all-remote company, with team members spread across 4 continents and 8 time zones.  The wider team of contributors from [all over the world](https://github.com/fleetdm/fleet/graphs/contributors) submit patches, bug reports, troubleshooting tips, improvements, and real-world insights to Fleet's open source code base, documentation, website, and company handbook.

### Open source
The majority of the code, documentation, and content we create at Fleet is public and source-available, and we strive to be broadly open and transparent in the way we run the business; as much as confidentiality agreements (and time) allow. We perform better with an audience, and our audience performs better with us.

### Direct responsibility
We use the concept of [directly responsible individuals](https://fleetdm.com/handbook/people#directly-responsible-individuals) (DRIs) to know who is responsible for what.  Every group maintains their own dedicated [handbook page](https://fleetdm.com/handbook), which is kept up to date with accurate, current information, including the group's [kanban board](https://github.com/orgs/fleetdm/projects?type=beta), Slack channels, and recurring tasks ("rituals").

#### Group Slack channels
Every group at Fleet maintains certain Slack channels, which all group members join and keep unmuted.  Everyone else at Fleet is encouraged to mute these channels, using them only as needed.  Each channel has a directly responsible individual who is responsible for keeping up with all new messages, even if they aren't explicitly mentioned (`@`).

## Things new and old team members should know
### Why do we wireframe first?
- Wireframing is called “drafting” at Fleet and is done in Figma.
- Anyone can make a wireframe suggestion, and wireframes are easy to contribute without being code literate.
- Drafting is completed for each change.
- We can throw it away after changes. Coding first leaves verbiage that is difficult to update, if it ever gets done at all.
- It allows you to simplify the creation and testing of error messages.
- Iterating in wireframes first lets us do all this for:
    - Error messages
    - Layouts
    - Flows
    - Interactions
    - Help text
    - Button text
    - Forms
    - URLs
    - API parameters
    - API response data…and more

### Why mono repo?

- One repo keeps all of the relevant work in one place. The only exception is when working on something confidential.
- One repo means that there is less to get lost.
- One repo pools GitHub stars to reflect Fleet’s actual presence better.

### Why organize work in team-based kanban boards?

- Kanban boards provide a uniform layout across all teams where anyone in the company can look to see what other teams are working on and have coming up.
- The different columns on the boards allow us to create a game plan for our to-do list for each 3 week iteration.
- These boards allow anyone in the world to contribute.

### Why 3 week cadence?

- Fleet product is released every 3 weeks, so everyone in the company is synced up to this same schedule.
- Other companies use a 4 week release cycle, but at Fleet, we like to move a little faster to get more done.
- Everyone always knows when the new release is, so they also know when their work is due.

### Why agile?

- See [the agile manifesto](https://agilemanifesto.org/).
- Collaborating and pushing for the next release creates the best product and culture.

Our values and mission.

- See [Fleet's values](./company.md#values).

### Why the emphasis on training?

- Investing in people makes them better and faster contributors.
- Creating a culture of helping others results in people feeling more comfortable and confident even if they aren’t familiar with osquery.
- A sharp focus on training means things are written down.

### Why handbook-first strategy?

- Watch [this video about the handbook-first strategy](https://www.youtube.com/watch?v=aZrK8AQM8Ro).
- For more details, see [GitLab’s handbook-first strategy](https://about.gitlab.com/company/culture/all-remote/handbook-first-documentation/).
- Documenting in the handbook allows Fleet to scale up and retain knowledge for consistency.

### Why not continuously generate REST API docs from javadoc-style code comments?
- It looks cheap. Those using open API still are embarrassed by their docs.
- Generated documentation via tools like Swagger/OpenAPI has a tendency to get out of date and becomes harder to fix to make it up to date.
- There is less control over how to add annotations to the doc.
- It has less visibility/ accessibility/ modifiability for people without Golang coding experience.
- Fully integrating with swagger's format sufficiently to document everything involves more people on the team learning about the intricacies of swagger (instead of editing in markdown that looks like any other markdown in the docs/website)).
- Autogenerating docs is not the only way to make sure docs accurately reflect the API.
- Generated docs become just as out of date as handmade docs, except since they are generated, they become more difficult to edit and therefore gated/siloed. Adaptability is efficient.
- Using markdown allows anyone to edit our docs.
- Replacing markdown files with code comments makes API reference docs harder to locate and edit.



## 🌈 Values

Fleet's values are a set of five ideals adopted by everyone on the team.  They describe the culture we are working together to deliver, inside and outside the company:

1. 🔴 Empathy
2. 🟠 Ownership
3. 🟢 Balance
4. 🔵 Objectivity
5. 🟣 Openness

When a new team member joins Fleet, they adopt the values, from day 1.  This way, even as the company grows, everybody knows what to expect from the people they work with.  Having a shared mindset keeps us quick and determined.

### 🔴 Empathy
Empathy leads to better understanding, better communication, and better decisions.  Try to understand what people may be going through, so you can help make it better.

- be customer first
  <!-- TODO: Figure out what to do with this commented-out bit.  I wrote it, but it's too long.  Maybe just delete it. (mikermcneil, feb 26, 2022)

  > #### Customer first
  > At Fleet, we think about the customer first.  No matter what kind of work you are doing, you serve the customer.
  > 
  > When customers buy Fleet, they trust us to provide the solution for important problems.
  > 
  > Imagine you are the person making the decision to buy Fleet.  Imagine you are responsible for all of your organization's laptops and servers.  Imagine you log in to the product every day.  Imagine you are the developer writing code to integrate with Fleet's REST API.  Imagine you are deploying the server, or running the upgrade scripts.  Imagine you are responsible for keeping your organization's computers secure and running smoothly.
  > 
  > You would rest easier, knowing that everyone who works at Fleet is seeking to deliver the experience they would want for themselves, in your shoes. -->
- consider your counterpart
  - for example: customers, contributors, colleagues, the other person in your Zoom meeting, the other folks in a Slack channel, the people who use software and APIs you build, the people following the processes you design.
  - ask questions like you would want to be asked
  - assume positive intent
  - be kind
  - quickly review pending changes where your review was requested <!-- TODO: (when you are requested as a reviewer in GitHub, respond quickly.  If pull requests start to stack up, merge conflicts can arise, or the original author can forget, or lose context for why they were making the change.  The more pending changes there are, the harder it is to sort through what needs to be reviewed next.) -->
  - be punctual
  - end meetings on time
- role play as a user
  - don't be afraid to rely on your imagination to understand <!-- TODO: (When making changes, put yourself in the mindset of the end user. Keep in mind how someone might use the product or process you're building for the first time, or how someone accustomed to the old way might react to a new change.) -->
  - developers are users too (REST API, fleetctl, docs)
  - contributor experience matters (but product quality and commitments come first)
  - bugs cause frustrating experiences and alienate users
  - patch with care (upgrading to new releases of Fleet can be time-consuming for users running self-managed deployments) <!-- TODO: (patch releases are important for improving security, quality, and stability. Cut a patch release if there is a security concern, previously stable features are unusable, or if a new feature advertised in the current release is unusable.  But remember that people have to actually install these updates!) -->
  - confusing error messages make people feel helpless, and can fill them with despair
  - error messages deserve to be good (it's worth it to spend time on them)
  - UI help text and labels deserve to be good (it's worth it to spend time on them)
- hospitality
  - "be a helper"   -mr rogers
  - think and say [positive things](https://www.theatlantic.com/family/archive/2018/06/mr-rogers-neighborhood-talking-to-kids/562352/)
  - use the `#thanks` channel to show genuine gratitude for other team member's actions
  - talking with users and contributors is time well spent
  - embrace the excitement of others (it's contagious)
  - make small talk at the beginning of meetings
  - be generous (go above and beyond; for example, the majority of the features Fleet releases [will always be free](https://fleetdm.com/pricing))
  - apply customer service principles to all users, even if they never buy Fleet
  - be our guest
- better humanity


### 🟠 Ownership

<!-- TODO: short preamble -->

- take responsibility
  - think like an owner
  - follow through on commitments (actions match your words)
  - own up to mistakes
  - understand why it matters (the goals of the work you are doing)
  - consider business impact (fast forward 12 months, consider total cost of ownership over the eternity of maintenance)
  - do things that don't scale, sometimes
- be responsive
  - respond quickly, even if you can't take further action at that exact moment
  - when you disagree, give your feedback; then agree and commit, or disagree and commit anyway
  - prefer short calls to long, asynchronous back and forth discussions in Slack
  - procrastination is a symptom of not knowing what to do next (if you find yourself avoiding reading or responding to a message, schedule a Zoom call with the people you need to figure it out)
- we win or lose together
  - think about the big picture, beyond your individual team's goals
  - success == creating value for customers
  - you're not alone in this - there's a great community of people able and happy to help
  - don't be afraid to spend time helping users, customers, and contributors (including colleagues on other teams)
  - be proactive: ask other contributors how you can help, regardless who is assigned to what
  - get all the way done; help unblock team members and other contributors to deliver value  <!-- TODO: (collaborate; help teammates see tasks through to completion) -->
- take pride in your work
  - be efficient   (your time is valuable, your work matters, and your focus is a finite resource; it matters how you spend it)
  - you don't need permission to be thoughtful
  - reread anything you write for users <!-- TODO: (Check everything that a user might read for clarity, spelling errors, and to make sure that it provides value.) -->
  - take your ideas seriously (great ideas come from everyone; write them out and see if they have merit)
  - think for yourself, from first principles
  - use reason (believe in your brain's capacity to evaluate a solution or idea, regardless of how popular it is)
  - you are on a hero's journey (motivate yourself intrinsically with self-talk; even boring tasks are more motivating, fun, and effective when you care)
- better results

### 🟢 Balance
Between overthinking and rushing, there is a [golden mean](https://en.wikipedia.org/wiki/Golden_mean_%28philosophy%29).

- iterate
  - baby steps <!-- TODO: (look for ways to make the smallest, minimally viable change. Small changes provide faster feedback, and help us to stay focused on quality) -->
  - pick low-hanging fruit (deliver value quickly where you can)
  - think ahead, then make the right decision for now
  - look before you leap (when facing a non-trivial problem, get perspective before you dive in; what if there is a simpler solution?) <!-- TODO: When facing a (non-trivial) problem, take a step back before diving into fixing it - put the problem back in context, think about the actual goal and not just the issue itself, sometimes the obvious solution misses the end goal, sometimes a simpler solution will emerge, or it may just confirm that the fix is the right one and you can go ahead with better confidence -->
- move quickly
  - "everything is in draft"
  - think, fast (balance thoughtfulness and planning with moving quickly)
  - aim to deliver daily
  - move quicker than 90% of the humans you know
  - resist gold-plating; avoid bike-shedding
- less is more
  - focus on fewer tasks at one time  <!-- TODO: (By focusing on fewer tasks at once, we are able to get more done, and to a higher standard, while feeling more positive about our work in the process.) -->
  - "boring solutions"
  - finish what you start, or at least throw it away loudly in case someone wants it
  - keep it simple (prioritize simplicity; people crave mental space in design, collaboration, and most areas of life) <!-- reduce cognitive load -->
  - use fewer words (lots of text == lots of work)
  - as time allows  ("I would have written a shorter letter, but I did not have the time." -Blaise Pascal)
- make time for self-care
  - to help you bring your best self when communicating with others, making decisions, etc
  - consider taking a break or going for a walk
  - take time off; it is better to have 100% focus for 80% of the time than it is to have 80% focus for 100% of the time
  - think about how to best organize your day/work hours to fit your life and maximize your focus
- better focus


### 🔵 Objectivity
<!-- TODO: write short preamble, like the others --> 

- be curious
  - ask great questions & take the time to truly listen
  - listen intently to feedback, and genuinely try to understand (especially constructive criticism)  <!-- TODO: Trust the feedback from counterparts. It’s easy to quickly say “no” or ignore feedback because we’re busy and we often default to our way of thinking is right. Trust that your counterpart is making a good suggestion and give it the time/consideration it deserves. -->
  - see failure as a beginning (it is rare to get things right the first time)
  - question yourself ("why do I think this?")
- underpromise, overdeliver
  - quality results often take longer than we anticipate
  - be practical about your limits, and about what's possible with the time and resources we have
  - be thorough (don't settle for "the happy path"; every real-world edge case deserves handling)
- prioritize truth (reality)
  - be wrong, show your work (it's better to make the right decision than it is to be right)
  - "strong opinions, loosely held"  (proceed boldly, but change your mind in the face of new evidence)
  - avoid sunk cost fallacy (getting attached to something just because you invested time working on it, or came up with it)
  - be fair to competitors ("may the best product win.")
  - give credit where credit is due; don't show favoritism <!-- as it breeds resentment, destroys employee morale, and creates disincentives for good performance.  Seek out ways to be fair to everyone - https://about.gitlab.com/handbook/values/#permission-to-play -->
  - facts, over commentary
- speak computer to computers
  - a lucky fix without understanding does more harm than good
  - when something isn't working, use the scientific method
  - especially when there is a bug, or when something is slow, or when a customer is having a problem
  - assume it's your fault
  - assume nothing else
- better rigour

### 🟣 Openness
<!-- TODO: preamble -->

- anyone can contribute
  - be outsider-friendly, inclusive, and approachable
  - [use small words](http://www.paulgraham.com/writing44.html) so readers understand more easily
  - prioritize accessible terminology and simple explanations to provide value to the largest possible audience of users
  - avoid acronyms and idioms which might not translate
  - welcome contributions to your team's work, from people inside or outside the company
  - get comfortable letting others contribute to your domain
  - believe in everyone
- write things down
  - "handbook first"
  - writing it down makes it real and allows others to read on their own time (and in their own timezone)
  - never stop consolidating and deduplicating content (gradually, consistently, bit by bit)
- embrace candor
  - "short toes" (don't be afraid of stepping on toes)
  - don't be afraid to speak up (ask questions, be direct, and interrupt)
  - give pointed, respectful feedback <!-- (in the same way you would want to receive it) -->
  - take initiative in trying to improve things (no need to wait [for consensus](https://twitter.com/ryanfalor/status/1182647229414166528?s=12))
  - communicate openly (if you think you should send a message to communicate something, send it; but keep comments brief and relevant)
- be transparent
  - "public by default"
  - build in the open
  - declassify with care (easier to overlook confidential info when declassifying vs. when changing something that is already public from the get-go)
  - [open source is forever](https://twitter.com/mikermcneil/status/1476799587423772674)
- better collaboration



## History

### 2014: Origins of osquery
In 2014, our CTO Zach Wasserman, together with [Mike Arpaia](https://twitter.com/mikearpaia/status/1357455391588839424) and the rest of their team at Facebook, created an open source project called [osquery](https://osquery.io).

### 2016: Origins of Fleet v1.0
A few years later, Zach, Mike Arpaia, and [Jason Meller](https://honest.security) founded [Kolide](https://kolide.com) and created Fleet: an open source platform that made it easier and more productive to use osquery in an enterprise setting.

### 2019: The growing community
When Kolide's attention shifted away from Fleet and towards their separate, user-focused SaaS offering, the Fleet community took over maintenance of the open source project. After his time at Kolide, Zach continued as lead maintainer of Fleet.  He spent 2019 consulting and working with the growing open source community to support and extend the capabilities of the Fleet platform.

### 2020: Fleet was incorporated
Zach partnered with our CEO, Mike McNeil, to found a new, independent company: Fleet Device Management Inc.  In November 2020, we [announced](https://medium.com/fleetdm/a-new-fleet-d4096c7de978) the transition and kicked off the logistics of moving the GitHub repository.



<meta name="maintainedBy" value="mikermcneil">
