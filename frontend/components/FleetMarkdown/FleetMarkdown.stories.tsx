import { Meta, StoryObj } from "@storybook/react";

import FleetMarkdown from "./FleetMarkdown";

const TestMarkdown = `
# Test Markdown

## This is a heading

### This is a subheading

#### This is a subsubheading


---
**bold**

*italic*

[test link](https://www.fleetdm.com)

- test list item 1
- test list item 2
- test list item 3

> test blockquote

\`code text\`
`;

const meta: Meta<typeof FleetMarkdown> = {
  title: "Components/FleetMarkdown",
  component: FleetMarkdown,
  args: { markdown: TestMarkdown },
};

export default meta;

type Story = StoryObj<typeof FleetMarkdown>;

export const Basic: Story = {};
