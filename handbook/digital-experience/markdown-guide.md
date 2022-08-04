# Markdown guide

## Headings

Try to stay within three or four heading levels. Complicated documents may use more, but pages with a simpler structure are easier to read.

| Markdown | Rendered heading |
|:--------------------|:-----------------------------|
| `# Heading 1` | <h1>Heading 1</h1> |
| `## Heading 2` | <h2>Heading 2</h2> |
| `### Heading 3` | <h3>Heading 3</h3> |
| `#### Heading 4` | <h4>Heading 4</h4> |

## Emphasis

| Markdown | Rendered text |
|:--------------------|:-----------------------------|
| `**Bold**` | <strong>Bold</strong> |
| `*Italic*` | <em>Italic</em> |
| `***Bold italic***` | <em><strong>Bold italic</strong></em> |
| `~~Strikethrough~~` | <s>Strikethrough</s> |

## Lists

### Ordered lists

| Markdown | Rendered list |
|:-------------|:-----------------------------|
| <pre>1. Line one<br>2. Line two  <br>3. Line three<br>4. Line four</pre> | 1. Line one<br>2. Line two<br> 3. Line three<br>4. Line four |
| <pre>1. Line one<br>1. Indent one<br>2. Line two<br>3. Line three<br>1. Indent one<br>2. Indent two<br>4. Line four</pre> | 1. Line one<br>&nbsp;1. Indent one<br>2. Line two<br>3. Line three<br>&nbsp;1. Indent one<br>&nbsp;2. Indent two<br>4. Line four |

### Unordered lists

| Markdown | Rendered list |
|:-------------|:-----------------------------|
| <pre>- Line one<br>- Line two  <br>- Line three<br>- Line four</pre> | - Line one<br>- Line two<br>- Line three<br>- Line four |
| <pre>- Line one<br> - Indent one<br>- Line two<br>- Line three<br> - Indent one<br> - Indent two<br>- Line four</pre> | - Line one<br>&nbsp;- Indent one<br>- Line two<br>- Line three<br>&nbsp;- Indent one<br>&nbsp;- Indent two<br>- Line four |

## Links

The Fleet website currently supports the following Markdown link types.

### Inline link

It's a classic.

#### Example

`[This is an inline link](https://domain.com/example.md)`

#### Rendered output

[This is an inline link](https://domain.com/example.md)

### Link with a tooltip

Adding a tooltip to your link is a great way to provide additional information.

#### Example

`[This is link with a tooltip](https://domain.com/example.md "You're awesome!")`

#### Rendered output

[This is link with a tooltip](https://domain.com/example.md "You're awesome!")

### URLs

Add angle brackets "< >" around a URL to turn it into a link.

#### Example

`<https://fleetdm.com>`

#### Rendered output

<https://fleetdm.com>

### Emails

To create a mailto link... oh wait, I'm not going to tell you.

> **Important**: To avoid spam, we **NEVER** use mailto links.

## Tables

To create a table, start with the header by separating rows with pipes (" | ").
Use dashes (at least 3) to separate the header, and add colons to align the text in the table columns.

#### Example

```
| Category one | Category two | Category three |
|:---|---:|:---:|
| Left alignment | Right alignment | Center Alignment |
```

#### Rendered output

| Category one | Category two | Category three |
|:---|---:|:---:|
| Left alignment | Right alignment | Center Alignment |

## Blockquotes

To add a tip blockquote, start a line with ">" and end the blockquote with a blank newline.

### Tip blockquotes

Tip blockquotes are the default blockquote style in our Markdown content.

#### Example

```
> This is a tip blockquote.
This line is rendered inside of the tip blockquote.

This line is rendered outside of the tip blockquote.
```

#### Rendered output

> This is a tip blockquote.
This line is rendered inside of the tip blockquote.

This line is rendered outside of the tip blockquote.

### Quote blockquotes

To add a quote blockquote, start the blockquote with "quote" in an HTML comment.

#### Example

```
><!-- quote --> This is a quote blockquote.
>Lines starting with an opening angle bracket (>) will be rendered on a different line in the blockquote.
```

#### Rendered output

><!-- quote --> This is a quote blockquote.
>Lines starting with an opening angle bracket (>) will be rendered on a different line in the blockquote.

### Large quote blockquote

You can add a large quote blockquote by starting a blockquote with "large-quote" in an HTML comment.

#### Example

```
><!-- large-quote --> This is a large blockquote.
>If the content of your quote blockquote is more than three sentences long, you can use a large quote blockquote to reduce the font size and line height of the rendered quote.
```

#### Rendered output

><!-- large-quote --> This is a large blockquote.
>If the content of your quote blockquote is more than three sentences long, you can use a large quote blockquote to reduce the font size and line height of the rendered quote.


<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Markdown-guide">