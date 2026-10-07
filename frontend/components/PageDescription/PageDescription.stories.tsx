import { Meta, StoryObj } from "@storybook/react";
import React from "react";

import CustomLink from "components/CustomLink";

import PageDescription from ".";

const meta: Meta<typeof PageDescription> = {
  component: PageDescription,
  title: "Components/PageDescription",
  argTypes: {
    variant: {
      control: { type: "select" },
      options: [undefined, "card", "tab-panel", "right-panel", "modal"],
    },
    content: { control: "text" },
    className: { control: "text" },
  },
  args: {
    content:
      "Descriptive text that sits under a page title and orients the reader.",
  },
};

export default meta;

type Story = StoryObj<typeof PageDescription>;

export const Page: Story = {};

export const Card: Story = {
  args: { variant: "card" },
};

export const TabPanel: Story = {
  args: { variant: "tab-panel" },
};

export const RightPanel: Story = {
  args: { variant: "right-panel" },
};

export const Modal: Story = {
  args: { variant: "modal" },
};

export const RichContent: Story = {
  args: {
    content: (
      <>
        Supports <strong>rich</strong> content, including{" "}
        <CustomLink url="https://fleetdm.com" text="links" newTab />.
      </>
    ),
  },
};
