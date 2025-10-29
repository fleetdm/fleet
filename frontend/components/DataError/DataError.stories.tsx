import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import DataError from "./DataError";

const meta: Meta<typeof DataError> = {
  title: "Components/Error Messages/Data error",
  component: DataError,
};

export default meta;

type Story = StoryObj<typeof DataError>;

export const Basic: Story = {};

export const WithChildren: Story = {
  args: {
    children: <p>this is custom JSX</p>,
  },
};
