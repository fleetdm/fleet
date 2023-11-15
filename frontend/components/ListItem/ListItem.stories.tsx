import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import Button from "components/buttons/Button";

import ListItem from "./ListItem";

const meta: Meta<typeof ListItem> = {
  title: "Components/ListItem",
  component: ListItem,
  args: {
    graphic: "file-configuration-profile",
    title: "List Item Title",
    details: (
      <>
        <span>Details </span>
        <span> &bull; </span>
        <span> more details</span>
      </>
    ),
    actions: (
      <>
        <Button>button 1</Button>
        <Button>Button 2</Button>
      </>
    ),
  },
};

export default meta;

type Story = StoryObj<typeof ListItem>;

export const Basic: Story = {};
