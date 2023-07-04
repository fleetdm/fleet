import { Meta, StoryObj } from "@storybook/react";

import LinkWithContext from "./LinkWithContext";

const meta: Meta<typeof LinkWithContext> = {
  title: "Components/LinkWithContext",
  component: LinkWithContext,
  args: {
    className: "link-with-context",
    children: "Link with context",
    to: "/",
    withParams: {
      type: "query",
      names: ["apples", "bananas"],
    },
    currentQueryParams: {
      apples: "1",
      bananas: "2",
    },
  },
};

export default meta;

type Story = StoryObj<typeof LinkWithContext>;

export const Basic: Story = {};
