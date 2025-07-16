# ADR-0002: Not using GitHub Discussions 💬

## Status 🚦

Proposed

## Date 📅

2025-07-16

## Context 🔍

Our team faces a challenge with managing complex technical discussions related to GitHub issues:

* 🗣️ GitHub issues are not conducive to having several complex discussions on multiple aspects of a story
* 💬 These discussions frequently happen in Slack instead
* ⏰ We lose all history in Slack after 90 days
* 🔍 There is no way to find the Slack discussion from the issue later
* 🌐 We are public by default, but Slack discussions are not public

This leads to loss of valuable context and decision-making history that could help future contributors understand why certain choices were made.

We initially considered GitHub Discussions as a potential solution to preserve these conversations and link them to issues.

## Decision ✅

We have decided **not to use GitHub Discussions** for managing complex technical conversations.

This decision was made after evaluating GitHub Discussions and discovering several blocking issues that prevent it from solving our core problems:

* 🔗 **Poor integration with GitHub Issues:** There is no standard linkage between Discussions and Issues, making it difficult to maintain context between the two systems.
* 🚫 **No Slack notifications:** We cannot get GitHub Discussion notifications in Slack, which is critical for our workflow:
  * Toast does not support GitHub Discussions
  * GitHub Slack plugin is too noisy for our needs
  * We could not find any other suitable integrations
  * Using a Slack webhook is also too noisy since it cannot DM a person and simply publishes to a channel
  * Building our own custom integration would be too time-consuming

## Consequences 🎭

**Benefits:** ✨

* 🎯 Avoids introducing a tool that doesn't solve our core problems
* ⏱️ Saves time that would be spent on custom integration development
* 🔄 Prevents workflow fragmentation across too many platforms

**Drawbacks:** ⚠️

* 📉 We still lose valuable discussion history after 90 days in Slack
* 🤷 No clear path forward for preserving complex technical discussions
* 🔍 Future contributors will continue to lack context on decision-making

**Impact:** 💫

* 🚦 No change to current workflows
* 🔎 Team will continue searching for a better solution

**Future considerations:** 🔮

* 🔍 Continue evaluating other tools and platforms for discussion preservation
* 💡 Consider alternative approaches such as:
  * 📝 Improving documentation practices to capture key decisions
  * 💭 Using issue comments more effectively
  * 📋 Creating design documents for complex features
  * 🔧 Exploring other discussion platforms that integrate better with our toolchain

## Alternatives considered 🤔

**Alternative 1: Use GitHub Discussions despite limitations**

* **Description:** Adopt GitHub Discussions and work around the notification issues
* **Pros:** Native GitHub feature, preserves discussions indefinitely, searchable
* **Cons:** Poor Slack integration breaks our workflow, requires constant manual checking, no standard linkage to issues
* **Reason not chosen:** The lack of Slack notifications would significantly disrupt our team's workflow and likely lead to missed discussions

**Alternative 2: Build custom Slack integration**

* **Description:** Develop our own integration to bridge GitHub Discussions and Slack
* **Pros:** Could provide exactly the notifications we need, customizable
* **Cons:** Significant development time, ongoing maintenance burden, diverts resources from core product
* **Reason not chosen:** Too time-consuming and would require ongoing maintenance

**Alternative 3: Use existing Slack webhook with modifications**

* **Description:** Configure Slack webhooks to notify about GitHub Discussions
* **Pros:** No custom development needed, quick to implement
* **Cons:** Too noisy (publishes to channels only), cannot DM individuals, no filtering options
* **Reason not chosen:** Would create too much noise in channels and doesn't support targeted notifications

## References 📖

* [GitHub Discussions Documentation](https://docs.github.com/en/discussions)
* [Toast](https://www.toast.ninja/)
* [GitHub Slack App](https://github.com/integrations/slack)
