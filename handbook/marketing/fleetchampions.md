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
1. Interview customer
2. Write draft case study
3. Get approved by customer
4. Publish case study

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

4. ADd the quote in the `quote` field and fill in all infomration. All fields are required.

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
1. <To be described>

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
