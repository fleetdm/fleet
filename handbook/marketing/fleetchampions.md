### This page describes the Fleet Champions Community and associated processes

[Fleet champions deck](https://docs.google.com/presentation/d/1gvMl7M6Wqi9k1-HlvIo-DtbdDpKJdBsZs2ZniQ5UFJo/edit?slide=id.p#slide=id.p)
[Fleet champions list](https://docs.google.com/spreadsheets/d/1fMs7qeZ9Rme1yf9_n8GxfEPzrAxPcvB5yk0w78VY_0A/edit?gid=1407640772#gid=1407640772)

## Types of championship that a Fleet customer can help with

1. Logo usage on our website
2. [Customer testimonials](https://fleetdm.com/customers)
3. Press interview
4. Press quote
5. Peer review - Gartner
6. Peer review - G2
7. Presenter on a webinar
8. Present in a conference
9. Public case study

-----

## Process to enable customer participation in press interview
1. <To be described>

## Process to present at an industry conference
1. <To be described>

## Process to present in a webinar
1. <To be described>

## Process to publish a public case study
1. Interview CSM
2. Write draft case study
3. Get approved by CSM
4. Publish case study

## Process to publish an anonymous case study
1.
Follow this process to publish a case study without identifying the customer. Anonymous case studies help share real outcomes while protecting the customer’s identity.

## Process

1. **Gather customer information using Momentum**

   Run the following prompt in Momentum through the company's #acc Slack channel:

   > “@ Momentum based on all the previous call recordings and meeting notes, answer what's possible from this list of questions to the best of your ability:.”

   Include these questions:

   - What’s your company’s name? 
   - What industry does your company operate in? Approximately how many endpoints (Mac, Windows, Linux, etc.) do you currently manage? 
   - What was the primary frustration or limitation with your previous tool (e.g., Jamf, Intune, Kandji) that led you to look for a change? 
   - Before Fleet, were there "blind spots" in your infrastructure (like Linux servers or remote laptops) that you couldn't see or manage effectively? 
   - What were the top 3 requirements Fleet had to meet during your evaluation (e.g., On-premise hosting, GitOps workflows, osquery integration)? 
   - How does Fleet’s open-source nature or transparency impact your confidence in the security of your device management stack? Fleet emphasizes "transparency" for end-users. 
   - How has this affected the trust or relationship between your IT team and your employees? 
   - How long did it take to migrate your fleet? What was the impact on your end-users during that transition? 
   - Can you estimate the savings in licensing costs or the percentage reduction since consolidating to Fleet? 
   - Can you share a specific example of something you’ve automated with Fleet’s API that was difficult before? 
   - How has real-time visibility changed your response time to new vulnerabilities or compliance audits? 
   - If you were speaking to a CIO at a peer company, what is the single biggest reason you would tell them to switch to Fleet? 
   - How does having Fleet in place allow your team to be more effective moving forward? 
   - How important was it to manage macOS, Windows, and Linux from a single binary/API rather than maintaining three separate silos? 
   - Fleet allows you to stream telemetry directly to your own data tools. 
   - How has having direct/instant access to your device data changed how your security team monitors for threats? 
   - Did the choice between Fleet Cloud and self-hosting influence your decision, and why was that control important for your compliance or security needs?

2. **Review the responses**

   Confirm which questions Momentum answered.

   Document any questions that were not answered.

3. **Schedule a review call with the CSM**

   Schedule a 30 minute call with the customer success manager (CSM) who owns the account.

4. **Validate the information**

   Review the answers with the CSM.

   Confirm the information is accurate and safe to publish anonymously.

5. **Write the case study**

   Create the case study using the validated answers.

   Remove any information that could identify the customer, including:

   - Company name  
   - Unique internal tools or processes  
   - Specific geographic identifiers  
   - Any other identifiable details

6. **Publish the case study**

   Add the case study to the website.

   Use the **customer code name** listed in the **Fleet Champions Community spreadsheet**. This spreadsheet is the source of truth.

7. **Add the case study to the customers page**

   After publishing the case study, add a tile for it on the **Customers** page.

## Process to publish customer testimonials on the website
Spontaneous, nice things that customers say about Fleet in:
- Conversations
- Offical meetings
- Slack
- Social media posts
- Anywhere else

... should be captured for posterity in the [testimonials.yml](https://github.com/fleetdm/fleet/blob/c86ad041b2cbeb6ddeac08464ca6d1bf88af0aa5/handbook/company/testimonials.yml#L31) file

**Steps:**
1. Copy the good parts of the text or post and the information about the person being quoted.
2. Open the [testimonials.yml](https://github.com/fleetdm/fleet/blob/c86ad041b2cbeb6ddeac08464ca6d1bf88af0aa5/handbook/company/testimonials.yml#L31) file.
3. Scroll to the bottom of the `YAML` file and add a new block, e.g,

```
-
  quote: Fleet made the process of migrating fast, easy, and simple. 
  quoteAuthorName: John O'Nolan
  quoteAuthorJobTitle: Founder & CEO
  quoteLinkUrl: https://www.linkedin.com/in/johnonolan/
  quoteAuthorProfileImageFilename: testimonial-author-john-o'nolan-100x100@2x.png
  productCategories: [Device management]
```

4. Add the quote in the `quote` field and fill in all information. All fields are *required*.

5. The `productCategories` field should only be populated with one of the following:

```
Observability
Software management
Device management
```

6. See the top of the [testimonials.yml](https://github.com/fleetdm/fleet/blob/c86ad041b2cbeb6ddeac08464ca6d1bf88af0aa5/handbook/company/testimonials.yml#L31) file for this formatting information if this explanation is unclear.
7. If the testimonial is being added for immediate use on the website, adding the the new text block with the correct fields is sufficient.
8. If the testimonial is being saved in the `testimonials.yml` file but not published the block just added should be commented out so that the text is not processed by the website build. Your block should look like this:

```
# -
#   quote: Fleet made the process of migrating fast, easy, and simple. 
#   quoteAuthorName: John O'Nolan
#   quoteAuthorJobTitle: Founder & CEO
#   quoteLinkUrl: https://www.linkedin.com/in/johnonolan/
#   quoteAuthorProfileImageFilename: testimonial-author-john-o'nolan-100x100@2x.png
#   productCategories: [Device management]
```

9. There should be a Marketing ritual for processing the commented out spontaneous testimonials at some reasonable interval (quarterly, monthly, etc.) that is either captured here or in another spot in the Marketing handbook.
   
## Process to publish logo on website
1. <To be described>

## Process to publish quote on peer review site such as Gartner Peer Review or G2
Follow this process to collect a quote on a peer review site such as Gartner Peer Insights or G2. These reviews help potential customers hear directly from other IT teams using Fleet.

### Process

1. **Coordinate with the CSM**

   Reach out to the customer success manager (CSM) who owns the account.

   Ask them to schedule a 30 minute meeting with the customer.

   The meeting should include the Content Specialist and the CMO if they have not yet met the customer.

2. **Introduce the purpose of the meeting**

   During the meeting, spend the first 5–10 minutes introducing the goal.

   Explain what the Fleet Champions Community is and how customer feedback helps others evaluating device management tools.

3. **Explain the peer review program**

   Show the slide that explains peer reviews.

   Explain that the goal is to collect honest feedback from real Fleet users on trusted review platforms.

4. **Encourage multiple reviewers**

   Let the customer know that more than one person from their company can submit a review.

   Explain that completing the review typically takes less than 5–10 minutes.

5. **Share the review link**

   Send the Endpoint Management Tool review link during the meeting:

   https://gtnr.io/BZYrTASKq

   Make sure the customer uses the Endpoint Management Tool link, not the vulnerability assessment link.

   After the meeting, send the same link again in a direct follow-up message to the person you spoke with.

## Process to request C-Level meeting with customer
1. <To be described>

## Process to request interaction with an analyst
1. <To be described>

## Process to request participation in a customer advisory board (CAB)
1. <To be described>

## Process to request participation in a product advisory board (CAB)
1. <To be described>

## Process to request participation in a reference call with analyst (when submitting a Fleet submission to a requested report participation.  For example - Gartner Magic Quadrant)
1. <To be described>

## Process to request sales reference call
1. <To be described>

<meta name="maintainedBy" value="akuthiala">
<meta name="title" value="🫧 Fleet Champions">
