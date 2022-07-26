# Markdown

## Headings

| Markdown option one | Rendered heading |
|:--------------------|:-----------------------------|
| `# Heading 1` | <h1>Heading 1</h1> 
| `## Heading 2` | <h2>Heading 2</h2>
| `### Heading 3` | <h3>Heading 3</h3>
| `#### Heading 4` | <h4>Heading 4</h4>

## Emphasis

| Markdown option one | HTML | Rendered Text |
|:--------------------|:-----------------------------|:-----------------------------|
| `**Bold**` | ```<strong>Bold</strong> ``` | <strong>Bold</strong> 
| `*Italic*` |  ```<em>Italic</em> ``` | <em>Italic</em>
| `***Bold italic***` | ```<em><strong>Bold italic</strong></em> ``` | <em><strong>Bold italic</strong></em>
| `~~Strikethrough~~` | ```<s>Strikethrough</s> ``` | <s>Strikethrough</s>
|  | `<ins>Underline</ins>` | <ins>Underline</ins>


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
| `- Line one`  <br> `- Line two`  <br> `- Line three ` <br> `- Line four` | <ul><li>Line one</li><li>iLine two</li><li>iLine three</li><li>iLine four</li></ul> |
| `- 7\. What a lucky number.`  <br> `- It's nothing like 13.` |  <ul><li> 7\. What a lucky number.  </li><li> It's nothing like 13. </li></ul> |
|`- Line one`  <br><br> &nbsp;&nbsp;&nbsp;&nbsp;`Your paragraph goes here.`  <br><br>   `- Line two` | <ul><li> Line one  </li></ul> &nbsp;&nbsp;&nbsp;&nbsp; Your paragraph goes here <br> <br><ul><li> Line two. </li></ul> |


To nest lists inside of other unordered lists using Markdown by including four spaces before each item you desire to indent.

`- Line one `
   &nbsp;&nbsp;&nbsp;&nbsp; `- Indent one`
`- Line two`
`- Line three`
   &nbsp;&nbsp;&nbsp;&nbsp; `- Indent one`
  &nbsp;&nbsp;&nbsp;&nbsp;  `- Indent two`
`- Line four`

This is how it will render 

- Line one 
    - Indent one
- Line two
- Line three
    - Indent one
    - Indent two
- Line four

## title

`This is how you create a term with [a tooltip and a link ](https://fleetdm.com/handbook/brand#commonly-used-terms "add information to a term when someone hovers")`

This is how you make separate a link to reference elsewhere in the page.

`[1]: <https://fleetdm.com/> "Fleet can help you."`

## title

To create malito link out of a URL, incase it in angle brackets like so:

`<https://fleetdm.com>`

This will render as: <https://fleetdm.com>

The same concept works with email addresses.

## title

`<fake@fleetdm.com>`

This will render as:<fake@fleetdm.com>

> *Important* To avoid spam, we *NEVER* user mailto links in the handbook or docs.

## title

`Everyone's favorite device management company is **[Fleet](https://fleetdm.com/)**.`

Everyone's favorite device management company is **[Fleet](https://fleetdm.com/)**.

## title

`Everyone's favorite device management company is *[Fleet](https://fleetdm.com/)*.`

Everyone's favorite device management company is *[Fleet](https://fleetdm.com/)*.

## title

| Markdown | Rendered output |
|:-----------|------------------|
|```` `` Sometimes you need to talk about `code` in your Markdown. `` ````| <code>Sometimes you need to talk about `code` in your Markdown.</code> |

<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Markdown">
