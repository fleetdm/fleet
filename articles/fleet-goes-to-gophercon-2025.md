# Fleet goes to GopherCon 2025

At the end of August, Fleet joined hundreds of Gophers at GopherCon 2025 in New York. [Myself](https://github.com/iansltx), [Dante](https://github.com/dantecatalfamo), and
[Jahziel](https://github.com/jahzielv) spent the week attending sessions and connecting with the Go community.

<!-- TODO gophercon-2025-russ-cox.jpg alt=Fleeties posing with Russ Cox -->

We spent Tuesday meeting and mingling with fellow Gophers, while making sure to play our part in the upcoming [4.74 release](https://github.com/fleetdm/fleet/milestone/170) of Fleet...and, of course, taking in the sights (and slices) in NYC.

<!-- TODO gophercon-2025-pizza.jpg alt=When in NYC, you can't miss slices.Eat() -->

Wednesday kicked off with a keynote from Google's Cameron Balahan on how Go ties into the AI era, reinforcing a theme that started in workshops the day before and continued throughout the single-track remaining days of the conference.

According to Cameron, Go's type system lends itself to fast feedback when LLMs don't quite one-shot an agentic coding solution. At the same time, Go's relative simplicity, standard library, and built-in opinions provide both a consistent training data set (we're sure Fleet's codebase is in there somewhere) and guardrails for more consistent generated code. 

Another highlight was that robust developer tools and accurate documentation help both humans and AIs to build better code. Finally, [Go's MCP SDK](https://github.com/modelcontextprotocol/go-sdk) grounds agents in best practices for the current release of Go, counteracting LLMs' preference for living in the past.

<!-- TODO gophercon-2025-wednesday-ai-humans-similar-needs.jpg alt=The keynote made a point that AIs and humans have similar needs for good developer tooling -->

A standout presentation on the security front was from Google's Jess McClintock. She walked through software supply chain security and introduced Google's [CapsLock](https://github.com/google/capslock) tool for catching when a library suddenly interacts with new parts of the host it's running on in a new version. These capability changes match behavior for high-profile supply chain compromises; you want your xz library to faithfully compress files, rather than rummaging around in SSH authentication. 

The presentation also underscored that supply chain analysis needs to evaluate immutable built binaries rather than just source code, as adversaries can swap out library source code to cover their tracks. Supply chain integrity is important to Fleet, hence our work on [SLSA attestation](https://fleetdm.com/guides/fleet-software-attestation) earlier this year, so it was great to learn about more tooling to help us on that journey.

Another highlight was Alan Donovan's presentation on codebase modernization. In some cases these modernization steps can be performed by [automated tooling](https://pkg.go.dev/golang.org/x/tools/gopls/internal/analysis/modernize). For cases not covered by the `modernize` package, or for upgrading your own code to consistently use new-style function calls, the [inline](https://pkg.go.dev/golang.org/x/tools/internal/refactor/inline) refactoring package helps to implement the [strangler pattern](https://martinfowler.com/bliki/StranglerFigApplication.html) just a bit quicker.

A few more solid presentations, a rooftop social, and one night of sleep later, Patricio Whittingslow kicked off GopherCon's final day with an exploration of [Go (TinyGo, specifically) as an entire operating system](https://www.gophercon.com/agenda/session/1557395).

The next talks focused on instrumenting, tracing, and profiling, including a dive into critical path analysis, which applies to the mechanics of an entire project just as much as it does to running code.

A few presentations on MCP and a lunch break later, Thursday's round of lightning talks kicked off. These seven-minute presentations covered everything from using Single Static Assignment form for static analysis and compile-time assertion techniques, to tips on effectively building software with AI (don't ask someone else to be the first person to review code you prompted an agent to write!)...and using Neovim. 

Later in the day, we also got a peek at the [Green Tea garbage collector](https://siddharthav.medium.com/green-tea-garbage-collector-63233aa5a9b5) available as an experiment in Go 1.25, as well as a rundown of [recent improvements (including FIPS-140 certification and post-quantum cipher support) of Go's crypto libraries](https://www.gophercon.com/agenda/session/1557398).

<!-- TODO gophercon-2025-thursday-no-vulns.jpg alt=No bad news is great news for Go cryptography -->

As a first-time attendee of GopherCon (and first-time attendee of New York City outside an airport), it was fun seeing the Go community showcase improvements in the language from a conference center in Manhattan alongside other Fleeties. For those of y'all who we met there, thanks for being part of a memorable hallway track, and we're looking forward to turning our learnings into improvements in Fleet itself. Thanks so much to the organizers, speakers, and sponsors who made this year's conference happen!

<!-- TODO gophercon-2025-thursday-ian-times-square.jpg alt=The post author at Times Square, complete with Fleet swag -->

<meta name="category" value="articles">
<meta name="authorGitHubUsername" value="iansltx">
<meta name="authorFullName" value="Ian Littman">
<meta name="publishedOn" value="2023-09-10">
<meta name="articleTitle" value="Fleet goes to GopherCon 2025">
<meta name="description" value="Engineering Fleeties made it out to New York for the 2025 edition of GopherCon, and brought back learnings to make Fleet even better">
