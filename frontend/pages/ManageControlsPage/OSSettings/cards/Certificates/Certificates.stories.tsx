import { Meta, StoryObj } from "@storybook/react";

import Certificates from "./Certificates";

const meta: Meta<typeof Certificates> = {
  title: "Components/Certificates",
  component: Certificates,
};

export default meta;

type Story = StoryObj<typeof Certificates>;

export const Basic: Story = {};
