# Company

## About Fleet

Fleet Device Management Inc is an [open core company](https://www.heavybit.com/library/video/commercial-open-source-business-strategies/) that sells subscriptions that offer [more features and support](https://fleetdm.com/pricing) for Fleet and [osquery](https://osquery.io), the leading open source endpoint agent.

We are dedicated to:

- 🧑‍🚀 automating IT and security.
- 💍 reducing the proliferation of agents, and growing the adoption of osquery (one agent to rule them all).
- 🪟 privacy, transparency, and trust through open source software.
- 👁️ remaining the freshest, simplest source of truth for every kind of computing device and OS.
- 💻 building a better way to manage computers.


## Culture

### All remote
Fleet Device Management Inc. is an all-remote company, with team members spread across 4 continents and 8 time zones.  The broader team of contributors [worldwide](https://github.com/fleetdm/fleet/graphs/contributors) submits patches, bug reports, troubleshooting tips, improvements, and real-world insights to Fleet's open source code base, documentation, website, and company handbook.

### Open source
The majority of the code, documentation, and content we create at Fleet is public and source-available. We strive to be open and transparent in the way we run the business; as much as confidentiality agreements (and time) allow. We perform better with an audience, and our audience performs better with us.


## Why this way?

### Why do we wireframe first?
- Wireframing is called “drafting” at Fleet and is done in Figma.
- Anyone can make a wireframe suggestion, and wireframes are easy to contribute without being code literate.
- Drafting is completed for each change.
- We can throw it away after changes. Coding first leaves verbiage that is difficult to update, if it ever gets done at all.
- It allows you to simplify the creation and testing of error messages.
- Iterating in wireframes first lets us do all this for:
    - error messages.
    - layouts.
    - flows.
    - interactions.
    - help text.
    - button text.
    - forms.
    - URLs.
    - API parameters.
    - API response data…[and more.](https://github.com/fleetdm/fleet/issues/4821)

### Why mono repo?
- One repo keeps all of the relevant work in one place. The only exception is when we're working on something confidential.
- One repo means that there is less to get lost.
- One repo pools GitHub stars to reflect Fleet’s actual presence better.

### Why organize work in team-based kanban boards?
- Kanban boards provide a uniform layout across all teams where anyone in the company can look to see what other teams are working on and have coming up.
- The different columns on the boards allow us to create a game plan for our to-do list for each 3-week iteration.
- These boards allow anyone in the world to contribute.

### Why 3-week cadence?
- Fleet product is released every 3 weeks, so everyone in the company is synced up to this same schedule.
- Other companies use a 4-week release cycle, but at Fleet, we like to move a little faster to get more done.
- Everyone knows when the new release is, so they also know when their work is due.

### Why agile?
- See [the agile manifesto](https://agilemanifesto.org/).
- Collaborating and pushing for the next release creates the best product and culture.

### Why the emphasis on training?
- Investing in people makes them better and faster contributors.
- Creating a culture of helping others results in people feeling more comfortable and confident even if they aren’t familiar with osquery.
- A sharp focus on training means things are written down.

### Why handbook-first strategy?
- Watch [this video about the handbook-first strategy](https://www.youtube.com/watch?v=aZrK8AQM8Ro).
- For more details, see [GitLab’s handbook-first strategy](https://about.gitlab.com/company/culture/all-remote/handbook-first-documentation/).
- Documenting in the handbook allows Fleet to scale up and retain knowledge for consistency.

### Why not continuously generate REST API reference docs from javadoc-style code comments?
- Using Markdown allows anyone to edit Fleet's docs.
- Generated docs become just as out of date as handmade docs, except since they are generated, they can become more difficult to edit and therefore gated/siloed.
- Keeping the source of truth in code files confers less visibility/ accessibility/ modifiability for people without Golang coding experience.
- Code comments are more difficult to locate and edit than a single Markdown file.
- Autogenerating docs from code is not the only way to make sure reference docs accurately reflect the API.
- As the Fleet REST API, documentation, and tools mature, a more declarative format such as OpenAPI might become the source of truth, but only after investing in a format and processes to make it visible, accessible, and modifiable for all contributors.

### Why direct responsibility?
We use the concept of [directly responsible individuals](https://fleetdm.com/handbook/people#directly-responsible-individuals) (DRIs) to know who is responsible for what.  Every group maintains its own dedicated [handbook page](https://fleetdm.com/handbook), which is kept up to date with accurate, current information, including the group's [kanban board](https://github.com/orgs/fleetdm/projects?type=beta), Slack channels, and recurring tasks ("rituals").

#### Why group Slack channels?
Groups are organized around goals.  Connecting people with the same goals helps them produce better results by fostering freer communication.  While groups sometimes align with the organization chart, some groups consist of people who do not report to the same manager.    For example, [product groups](https://fleetdm.com/handbook/product) like `#g-agent` include engineers too, not just the product manager.

Every group at Fleet maintains specific Slack channels, which all group members join and keep unmuted.  Everyone else at Fleet is encouraged to mute these channels, using them only as needed.  Each channel has a directly responsible individual responsible for keeping up with all new messages, even if they aren't explicitly mentioned (`@`).



## 🌈 Values

Fleet's values are a set of five ideals adopted by everyone on the team.  They describe the culture we are working together to deliver, inside and outside the company:

1. 🔴 Empathy
2. 🟠 Ownership
3. 🟢 Balance
4. 🔵 Objectivity
5. 🟣 Openness

When a new team member joins Fleet, they adopt the values, from day 1.  This way, even as the company grows, everybody knows what to expect from the people they work with. Having a shared mindset keeps us quick and determined.

### 🔴 Empathy
Empathy leads to better understanding, better communication, and better decisions.  Try to understand what people may be going through, so you can help make it better.

- Be customer-first.
  <!-- TODO: Figure out what to do with this commented-out bit.  I wrote it, but it's too long.  Maybe just delete it. (mikermcneil, feb 26, 2022)

  > #### Customer first
  > At Fleet, we think about the customer first.  No matter what kind of work you are doing, you serve the customer.
  > 
  > When customers buy Fleet, they trust us to provide the solution for important problems.
  > 
  > Imagine you are the person making the decision to buy Fleet.  Imagine you are responsible for all of your organization's laptops and servers.  Imagine you log in to the product every day.  Imagine you are the developer writing code to integrate with Fleet's REST API.  Imagine you are deploying the server, or running the upgrade scripts.  Imagine you are responsible for keeping your organization's computers secure and running smoothly.
  > 
  > You would rest easier, knowing that everyone who works at Fleet is seeking to deliver the experience they would want for themselves, in your shoes. -->
- Consider your counterpart.
  - For example: customers, contributors, colleagues, the other person in your Zoom meeting, the other folks in a Slack channel, the people who use software and APIs you build, the people following the processes you design.
  - Ask questions as you would want to be asked.
  - Assume positive intent.
  - Be kind.
  - Quickly review pending changes where your review was requested. <!-- TODO: (when you are requested as a reviewer in GitHub, respond quickly.  If pull requests start to stack up, merge conflicts can arise, or the original author can forget, or lose context for why they were making the change.  The more pending changes there are, the harder it is to sort through what needs to be reviewed next.) -->
  - Be punctual.
  - End meetings on time.
- Role play as a user.
  - Don't be afraid to rely on your imagination to understand. <!-- TODO: (When making changes, put yourself in the mindset of the end user. Keep in mind how someone might use the product or process you're building for the first time, or how someone accustomed to the old way might react to a new change.) -->
  - Developers are users too (REST API, fleetctl, docs).
  - The contributor experience matters (but product quality and commitments come first).
  - Bugs cause frustrating experiences and alienate users.
  - Patch with care (upgrading to new releases of Fleet can be time-consuming for users running self-managed deployments). <!-- TODO: (patch releases are important for improving security, quality, and stability. Cut a patch release if there is a security concern, previously stable features are unusable, or if a new feature advertised in the current release is unusable.  But remember that people have to actually install these updates!) -->
  - Confusing error messages make people feel helpless and can fill them with despair.
  - Error messages deserve to be good (it's worth it to spend time on them).
  - UI help text and labels deserve to be good (it's worth it to spend time on them).
- Be hospitable. 
  - "Be a helper."   -Mr. Rogers
  - Think and say [positive things](https://www.theatlantic.com/family/archive/2018/06/mr-rogers-neighborhood-talking-to-kids/562352/).
  - Use the `#thanks` channel to show genuine gratitude for other team member's actions.
  - Talking with users and contributors is time well spent.
  - Embrace the excitement of others (it's contagious).
  - Make small talk at the beginning of meetings.
  - Be generous (go above and beyond; for example, the majority of the features Fleet releases [will always be free](https://fleetdm.com/pricing)).
  - Apply customer service principles to all users, even if they never buy Fleet.
  - Be our guest.
- Better humanity.


### 🟠 Ownership

<!-- TODO: short preamble -->

- Take responsibility.
  - Think like an owner.
  - Follow through on commitments (actions match your words).
  - Own up to mistakes.
  - Understand why it matters (the goals of the work you are doing).
  - Consider the business impact (fast forward 12 months, consider the total cost of ownership over the eternity of maintenance).
  - Do things that don't scale, sometimes.
- Be responsive.
  - Respond quickly, even if you can't take further action at that exact moment.
  - When you disagree, give your feedback; then agree and commit, or disagree and commit anyway.
  - Favor short calls to long, asynchronous back and forth discussions in Slack.
  - Procrastination is a symptom of not knowing what to do next (if you find yourself avoiding reading or responding to a message, schedule a Zoom call with the people you need to figure it out).
- We win or lose together.
  - Think about the big picture, beyond your individual team's goals
  - Success equals creating value for customers.
  - You're not alone in this (there's a great community of people able and happy to help).
  - Don't be afraid to spend time helping users, customers, and contributors (including colleagues on other teams).
  - Be proactive (ask other contributors how you can help, regardless of who is assigned to what).
  - Get all the way done (help unblock team members and other contributors to deliver value).  <!-- TODO: (collaborate; help teammates see tasks through to completion) -->
- Take pride in your work.
  - Be efficient (your time is valuable, your work matters, and your focus is a finite resource; it matters how you spend it).
  - You don't need permission to be thoughtful.
  - Reread anything you write for users. <!-- TODO: (Check everything that a user might read for clarity, spelling errors, and to make sure that it provides value.) -->
  - Take your ideas seriously (great ideas come from everyone; write them out and see if they have merit).
  - Think for yourself, from first principles.
  - Use reason (believe in your brain's capacity to evaluate a solution or idea, regardless of how popular it is).
  - You are on a hero's journey (motivate yourself intrinsically with self-talk; even boring tasks are more motivating, fun, and effective when you care).
- Better your results.

### 🟢 Balance
Between overthinking and rushing, there is a [golden mean](https://en.wikipedia.org/wiki/Golden_mean_%28philosophy%29).

- Remember to iterate.
  - Work in baby steps. <!-- TODO: (look for ways to make the smallest, minimally viable change. Small changes provide faster feedback, and help us to stay focused on quality) -->
  - Pick low-hanging fruit (deliver value quickly where you can).
  - Think ahead, then make the right decision for now.
  - Look before you leap (when facing a non-trivial problem, get perspective before you dive in; what if there is a simpler solution?). <!-- TODO: When facing a (non-trivial) problem, take a step back before diving into fixing it - put the problem back in context, think about the actual goal and not just the issue itself, sometimes the obvious solution misses the end goal, sometimes a simpler solution will emerge, or it may just confirm that the fix is the right one and you can go ahead with better confidence -->
- Move quickly.
  - "Everything is in draft."
  - Think, fast (balance thoughtfulness and planning with moving quickly).
  - Aim to deliver daily.
  - Move quicker than 90% of the humans you know.
  - Resist gold-plating and avoid bike-shedding.
- Less is more.
  - Focus on fewer tasks at one time.  <!-- TODO: (By focusing on fewer tasks at once, we are able to get more done, and to a higher standard, while feeling more positive about our work in the process.) -->
  - Go for "boring solutions."
  - Finish what you start, or at least throw it away loudly in case someone wants it.
  - Keep it simple (prioritize simplicity; people crave mental space in design, collaboration, and most areas of life). <!-- reduce cognitive load -->
  - Use fewer words (lots of text equals lots of work).
  - Complete tasks as time allows  ("I would have written a shorter letter, but I did not have the time." -Blaise Pascal).
- Make time for self-care.
  - This will help you bring your best self when communicating with others, making decisions, etc.
  - Consider taking a break or going for a walk.
  - Take time off; it is better to have 100% focus for 80% of the time than it is to have 80% focus for 100% of the time.
  - Think about how to best organize your day/work hours to fit your life and maximize your focus.
- Better your focus.


### 🔵 Objectivity
<!-- TODO: write short preamble, like the others --> 

- Be curious.
  - Ask great questions & take the time to listen truly.
  - Listen intently to feedback, and genuinely try to understand (especially constructive criticism).  <!-- TODO: Trust the feedback from counterparts. It’s easy to quickly say “no” or ignore feedback because we’re busy and we often default to our way of thinking is right. Trust that your counterpart is making a good suggestion and give it the time/consideration it deserves. -->
  - See failure as a beginning (it is rare to get things right the first time).
  - Question yourself ("why do I think this?").
- Underpromise and overdeliver.
  - Quality results often take longer than we anticipate.
  - Be practical about your limits, and about what's possible with the time and resources we have.
  - Be thorough (don't settle for "the happy path"; every real-world edge case deserves handling).
- Prioritize the truth (reality).
  - Be wrong and show your work (it's better to make the right decision than it is to be right).
  - Think "strong opinions, loosely held"  (proceed boldly, but change your mind in the face of new evidence)
  - Avoid the sunk cost fallacy (getting attached to something just because you invested time working on it, or came up with it).
  - Be fair to competitors ("may the best product win.").
  - Give credit where credit is due; don't show favoritism. <!-- as it breeds resentment, destroys employee morale, and creates disincentives for good performance.  Seek out ways to be fair to everyone - https://about.gitlab.com/handbook/values/#permission-to-play -->
  - Hold facts, over commentary.
- speak computer to computers
  - A lucky fix without understanding does more harm than good.
  - When something isn't working, use the scientific method.
  - Especially think like a computer when there is a bug, or when something is slow, or when a customer is having a problem.
  - Assume it's your fault.
  - Assume nothing else.
- Better  your rigor.

### 🟣 Openness
<!-- TODO: preamble -->

- Anyone can contribute to Fleet.
  - Be outsider-friendly, inclusive, and approachable.
  - [Use small words](http://www.paulgraham.com/writing44.html) so readers understand more easily.
  - Prioritize accessible terminology and simple explanations to provide value to the largest possible audience of users.
  - Avoid acronyms and idioms which might not translate.
  - Welcome contributions to your team's work, from people inside or outside the company.
  - Get comfortable letting others contribute to your domain.
  - Believe in everyone.
- Write everything down.
  - Use the "handbook first" strategy.
  - Writing your work down makes it real and allows others to read on their own time (and in their own timezone).
  - Never stop consolidating and deduplicating content (gradually, consistently, bit by bit).
- Embrace candor.
  - Have "short toes" and don't be afraid of stepping on toes.
  - Don't be afraid to speak up (ask questions, be direct, and interrupt).
  - Give pointed and respectful feedback. <!-- (in the same way you would want to receive it) -->
  - Take initiative in trying to improve things (no need to wait [for a consensus](https://twitter.com/ryanfalor/status/1182647229414166528?s=12)).
  - Communicate openly (if you think you should send a message to communicate something, send it; but keep comments brief and relevant).
- Be transparent.
  - Everything we do is "public by default."
  - We build in the open.
  - Declassify with care (easier to overlook confidential info when declassifying vs. when changing something that is already public from the get-go).
  - [Open source is forever](https://twitter.com/mikermcneil/status/1476799587423772674).
- Better your collaboration.



## History

### 2014: Origins of osquery
In 2014, our CTO Zach Wasserman, together with [Mike Arpaia](https://twitter.com/mikearpaia/status/1357455391588839424) and the rest of their team at Facebook, created an open source project called [osquery](https://osquery.io).

### 2016: Origins of Fleet v1.0
A few years later, Zach, Mike Arpaia, and [Jason Meller](https://honest.security) founded [Kolide](https://kolide.com) and created Fleet: an open source platform that made it easier and more productive to use osquery in an enterprise setting.

### 2019: The growing community
When Kolide's attention shifted away from Fleet and towards their separate, user-focused SaaS offering, the Fleet community took over maintenance of the open source project. After his time at Kolide, Zach continued as lead maintainer of Fleet.  He spent 2019 consulting and working with the growing open source community to support and extend the capabilities of the Fleet platform.

### 2020: Fleet was incorporated
Zach partnered with our CEO, Mike McNeil, to found a new, independent company: Fleet Device Management Inc.  In November 2020, we [announced](https://medium.com/fleetdm/a-new-fleet-d4096c7de978) the transition and kicked off the logistics of moving the GitHub repository.


## Slack channels

The following [Slack channels are maintained](https://fleetdm.com/handbook/company#group-slack-channels) by Fleet's founders and executive collaborators:

| Slack channel               | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:----------------------------|:--------------------------------------------------------------------|
| `#g-founders`               | Mike McNeil
| `#help-mission-control`     | Charlie Chance
| `#help-okrs`                | Mike McNeil
| `#help-manage`              | Mike McNeil
| `#news-fundraising`         | Mike McNeil
| `#help-open-core-ventures`  | Mike McNeil
| `#general`                  | N/A _(announce something company-wide)_
| `#thanks`                   | N/A _(say thank you)_
| `#random`                   | N/A _(be random)_

<meta name="maintainedBy" value="mikermcneil">
