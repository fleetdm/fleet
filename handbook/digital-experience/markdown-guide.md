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

<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Markdown-guide">