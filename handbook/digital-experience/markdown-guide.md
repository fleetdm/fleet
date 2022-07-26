# Markdown

## title

| Markdown option one | Rendered heading |
|:--------------------|:-----------------------------|
| `# Heading 1` | <h1>Heading 1</h1> 
| `## Heading 2` | <h2>Heading 2</h2>
| `### Heading 3` | <h3>Heading 3</h3>
| `#### Heading 4` | <h4>Heading 4</h4>

## title

| Markdown option one | HTML | Rendered Text |
|:--------------------|:-----------------------------|:-----------------------------|
| `**Bold**` | ```<strong>Bold</strong> ``` | <strong>Bold</strong> 
| `*Italic*` |  ```<em>Italic</em> ``` | <em>Italic</em>
| `***Bold italic***` | ```<em><strong>Bold italic</strong></em> ``` | <em><strong>Bold italic</strong></em>
| `~~Strikethrough~~` | ```<s>Strikethrough</s> ``` | <s>Strikethrough</s>
|  | `<ins>Underline</ins>` | <ins>Underline</ins>


## title

| Markdown | HTML | Rendered List |
|:-------------  |:---------------------------|:-----------------------------|
| `1. Line one`  <br> `2. Line two`  <br> `3. Line three ` <br> `4. Line four`   |``` <ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line one</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line two</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line three</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line four</li>``` <br> ```</ol>``` | 1. Line one  <br> 2. Line two  <br> 3. Line three  <br> 4. Line four
| `1. Line one` <br> `4. Line two` <br> `2. Line three` <br> `5. Line four`| ``` <ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line one</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line two</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line three</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line four</li>``` <br> ```</ol>``` | 1. Line one  <br> 2. Line two  <br> 3. Line three  <br> 4. Line four 
| `1. Line one` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`1. Indent one` <br> `2. Line two`  <br> `3. Line three` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; `1. Indent one`<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; `2. Indent two` <br> `4. Line four`   |``` <ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line one</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;``` <ol>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Indent one</li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line two</li>``` <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line three</li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<ol>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Indent one </li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; ```<li>Indent two</li>```<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;```<li>Line four</li>``` <br> ```</ol>``` | 1. Line one<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;1. Indent one <br> 2. Line two  <br> 3. Line three <br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; 1. Indent one<br>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; 2. Indent two <br> 4. Line four  

## title

`This is how you create a term with [a tooltip and no link ](## "add information to a term when someone hovers")`

This is how you create a term with [a tooltip and no link ](## "add information to a term when someone hovers").

`This is how you create a term with [a tooltip and a link ](https://fleetdm.com/handbook/brand#commonly-used-terms "add information to a term when someone hovers")`

This is how you create a term with [a tooltip and a link ](# "add information to a term when someone hovers").

This is how you make a tooltip with the link separated, at the bottom of the page.

`[1]: <https://en.wikipedia.org/wiki/Hobbit#Lifestyle> "Hobbit lifestyles"`

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
