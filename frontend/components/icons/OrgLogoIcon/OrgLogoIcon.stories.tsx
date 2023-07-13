import { Meta, StoryObj } from "@storybook/react";

// @ts-ignore
import OrgLogoIcon from ".";

const meta: Meta<typeof OrgLogoIcon> = {
  component: OrgLogoIcon,
  title: "Components/OrgLogoIcon",
  args: {},
};

export default meta;

type Story = StoryObj<typeof OrgLogoIcon>;

export const Default: Story = {};
