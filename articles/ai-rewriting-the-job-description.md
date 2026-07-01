# AI isn't just replacing jobs, it's rewriting the job description

When General Motors laid off more than 500 IT workers in May and immediately posted 80 open roles for AI-native developers and data engineers, TechCrunch called it a "skills swap." That framing is a little too neat. But it points at something real, and it's not just happening at GM.

The question worth asking isn't whether AI is disrupting IT employment. It is. The question is how, and whether the answer is what you think it is.

## The pattern in the data

A Stanford study published in August 2025 analyzed ADP payroll data across millions of workers in AI-exposed roles, including software engineering, data analysis, and IT operations. The results are more specific than the usual "AI takes jobs" narrative.

Workers aged 22 to 25 in AI-exposed roles saw a 13% relative employment decline since late 2022. Software developer employment in that cohort fell nearly 20% from its 2022 peak. Workers 30 and older in the same roles saw employment grow 6 to 12%.

The dividing line is not job function. It's experience, and specifically the kind of experience that makes AI a collaborator rather than a substitute. People who have accumulated deep systems knowledge, developed judgment about what matters in their environment, and learned to recognize when something technically correct is contextually wrong: those people are using AI to do more, faster. People who expected routine and repeatable work to build that foundation are finding it gone before they built it.

PwC's 2025 Global AI Jobs Barometer, which analyzed nearly a billion job postings across six continents, found a 56% wage premium for roles requiring AI skills, up from 25% the year before. The jobs exist. The required skills are changing faster than most people can keep up.

## What this looks like in IT and platform engineering

For IT teams specifically, the shift has a concrete shape.

AI tools are most useful when they have something structured to work against. A repository of consistent, legible configuration files is exactly that. Ask an AI to write a policy check, scope it to a specific set of devices, link it to a compliance control, and open a pull request against an existing GitOps repo: that loop closes in seconds. The AI reads the existing structure and produces output that fits it.

The same request against a GUI-based management system goes nowhere. There's no schema to learn from, no examples to pattern-match, no place for the output to land without a human translating it into clicks. AI tooling doesn't make ClickOps faster. It makes code-first workflows much faster. We've written more about this in [Why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops).

This creates a concrete skills divide. Client platform engineers who have moved to code-first device management workflows are positioned to get significantly more leverage from AI tools than those who haven't. The productivity gap between those teams is already measurable and is widening.

## The real downsides

The labor disruption is real, and the transition costs fall on individuals, not on the organizations making the decisions. GM reported strong earnings the same quarter it made those cuts. The productivity gains are going somewhere, and it isn't to displaced workers.

Technical trust in AI isn't as settled as the enthusiasm suggests, either. The Stack Overflow 2025 Developer Survey found that trust in AI accuracy among developers fell year over year, from 40% to 29%. People who use these tools every day are more skeptical of them than they were a year ago. That's worth sitting with.

None of this means the change isn't happening. It's a reason to approach it clearly, without catastrophizing or cheerleading.

## What positions you well

The World Economic Forum's Future of Jobs Report 2025 projects 170 million new jobs and 92 million displaced by 2030, a net gain with significant distributional unevenness. The DORA 2025 report is direct about what that means in practice: AI amplifies what's already there. Strong engineers get stronger. Weak processes get faster chaos.

The skill is not just using AI. It's knowing when, how, and whether, and having enough underlying judgment to catch it when it's wrong.

For IT teams and client platform engineers, a few things matter in concrete terms.

Get hands-on with AI in real work, not toy examples. Understand what these tools do well (syntax, boilerplate, querying structured schemas) and where they fail (organizational context, recognizing when technically correct is operationally wrong).

Build config that AI can work with. If your device management configuration lives in a GUI, you're not positioned to benefit from AI-assisted authoring. Structured, version-controlled config isn't only good engineering practice anymore. It's the surface AI needs to produce useful output.

Invest in judgment over syntax. The skills that hold their value are the ones AI doesn't substitute for well: knowing which policies actually matter for your environment, understanding your org's specific risk tolerance, designing systems that are reviewable and maintainable by people who weren't in the room when they were built. Those remain human problems.

If you're waiting to engage until you're sure this is real: the gap between teams moving now and teams moving later is already measurable. The job description is being rewritten. The question is whether you're involved in writing it.

<meta name="articleTitle" value="AI isn't just replacing jobs, it's rewriting the job description">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-06-26">
<meta name="description" value="AI isn't eliminating IT jobs. It's changing what IT work looks like, and the divide it's drawing runs through IT teams. Here's what the data shows.">
<meta name="category" value="articles">