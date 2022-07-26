# Markdown guide

## Headings

The fewer " # " the larger the heading.

| Markdown option one | Rendered heading |
|:--------------------|:-----------------------------|
| `# Heading 1` | <h1>Heading 1</h1> |
| `## Heading 2` | <h2>Heading 2</h2> |
| `### Heading 3` | <h3>Heading 3</h3> |
| `#### Heading 4` | <h4>Heading 4</h4> |

## Emphasis

| Markdown option one | HTML | Rendered Text |
|:--------------------|:-----------------------------|:-----------------------------|
| `**Bold**` | ```<strong>Bold</strong> ``` | <strong>Bold</strong> |
| `*Italic*` |  ```<em>Italic</em> ``` | <em>Italic</em> |
| `***Bold italic***` | ```<em><strong>Bold italic</strong></em> ``` | <em><strong>Bold italic</strong></em> |
| `~~Strikethrough~~` | ```<s>Strikethrough</s> ``` | <s>Strikethrough</s> |
|  | `<ins>Underline</ins>` | <ins>Underline</ins> |


## Ordered lists

| Markdown | HTML | Rendered List |
|:-------------  |:---------------------------|:-----------------------------|
| `1. Line one`  <br> `2. Line two`  <br> `3. Line three ` <br> `4. Line four`   |``` <ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line one</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line two</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line three</li>```  <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line four</li>``` <br> ```</ol>``` | 1. Line one  <br> 2. Line two  <br> 3. Line three  <br> 4. Line four|
| `1. Line one` <br> `4. Line two` <br> `2. Line three` <br> `5. Line four`| ``` <ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line one</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line two</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line three</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line four</li>``` <br> ```</ol>``` | 1. Line one  <br> 2. Line two  <br> 3. Line three  <br> 4. Line four |
| `1. Line one` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`1. Indent one` <br> `2. Line two`  <br> `3. Line three` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; `1. Indent one`<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; `2. Indent two` <br> `4. Line four`   |``` <ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line one</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;``` <ol>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Indent one</li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line two</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line three</li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Indent one </li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; ```<li>Indent two</li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line four</li>``` <br> ```</ol>``` | 1. Line one<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;1. Indent one <br> 2. Line two  <br> 3. Line three <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; 1. Indent one<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; 2. Indent two <br> 4. Line four  |

## Unordered lists

See the Markdown examples of unordered lists below.

| Markdown option one | Rendered List&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;|
|:-------------  |:--------------------------------------------|
| `- Line one`  <br> `- Line two`  <br> `- Line three ` <br> `- Line four` | <ul><li>Line one</li><li>Line two</li><li>Line three</li><li>Line four</li></ul> |
| `- 7\. What a lucky number.`  <br> `- It's nothing like 13.` |  <ul><li> 7\. What a lucky number.  </li><li> It's nothing like 13. </li></ul> |
|`- Line one`  <br><br> &nbsp;&nbsp;&nbsp;&nbsp;`Your paragraph goes here.`  <br><br>   `- Line two` | <ul><li> Line one  </li></ul> &nbsp;&nbsp;&nbsp;&nbsp; Your paragraph goes here <br> <br><ul><li> Line two. </li></ul> |

Nest lists inside other unordered lists using Markdown by including four spaces before each item you desire to indent.

`- Line one `
   &nbsp;&nbsp;&nbsp;&nbsp; `- Indent one`
`- Line two`
`- Line three`
   &nbsp;&nbsp;&nbsp;&nbsp; `- Indent one`
  &nbsp;&nbsp;&nbsp;&nbsp;  `- Indent two`
`- Line four`

This renders as 

- Line one 
    - Indent one
- Line two
- Line three
    - Indent one
    - Indent two
- Line four

## Links

Type **command + K** for a simple link template. 

`This is how you create a term with [a link ](https://fleetdm.com/handbook/brand#commonly-used-terms)`

This renders as

This is how you create a term with [a link ](https://fleetdm.com/handbook/brand#commonly-used-terms)

`This is how you create a term with [a tooltip and a link ](https://fleetdm.com/handbook/brand#commonly-used-terms "add information to a term when someone hovers")`

This renders as

This is how you create a term with [a tooltip and a link ](https://fleetdm.com/handbook/brand#commonly-used-terms "add information to a term when someone hovers")

This is how you make separate a link to reference elsewhere in the page.

`[1]: <https://fleetdm.com/> "Fleet can help you."`
`[This is a separated link](1)`

This renders as

[1]: <https://fleetdm.com/> "Fleet can help you."
[This is a separated link](1)

To create mailto link out of a URL, encase it in angle brackets like so:

`<https://fleetdm.com>`

This renders as

<https://fleetdm.com>

The same concept works with email addresses.

## Mailto links

Add angle brackets " < > " around a URL to turn it into a link.

`<fake@fleetdm.com>`

This renders as

<fake@fleetdm.com>

> *Important*: To avoid spam, we *NEVER* user mailto links in the handbook or docs.

<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Markdown-guide">
