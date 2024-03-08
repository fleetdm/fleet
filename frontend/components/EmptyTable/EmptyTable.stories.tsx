import React from "react";
import type { Meta, StoryObj } from "@storybook/react";

import Button from "components/buttons/Button";
import EmptyTable from "./EmptyTable";

const meta: Meta<typeof EmptyTable> = {
  title: "Components/EmptyTable",
  component: EmptyTable,
  argTypes: {
    className: {
      control: "text",
    },
  },
};

export default meta;

type Story = StoryObj<typeof EmptyTable>;

export const Basic: Story = {
  args: {
    header: "No Data",
    info: "There is no data to display.",
    graphicName: "empty-queries",
  },
};

export const WithAdditionalInfo: Story = {
  args: {
    ...Basic.args,
    additionalInfo: "You can add additional info here.",
  },
};

export const WithPrimaryButton: Story = {
  args: {
    ...WithAdditionalInfo.args,
    primaryButton: <Button>ok</Button>,
  },
};

export const WithSecondaryButton: Story = {
  args: {
    ...WithPrimaryButton.args,
    secondaryButton: <Button>cancel</Button>,
  },
};
