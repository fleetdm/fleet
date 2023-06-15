import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import Checkbox from ".";

const meta: Meta<typeof Checkbox> = {
  component: Checkbox,
  title: "Components/FormFields/Checkbox",
};

export default meta;

type Story = StoryObj<typeof Checkbox>;

export const Basic: Story = {
  parameters: {
    design: {
      type: "figma",
      url:
        "https://www.figma.com/file/qbjRu8jf01BzEfdcge1dgu/Fleet-style-guide-2022-(WIP)?node-id=117-16951",
    },
  },
};

export const WithLabel: Story = {
  args: {
    children: <b>Label</b>,
  },
};
