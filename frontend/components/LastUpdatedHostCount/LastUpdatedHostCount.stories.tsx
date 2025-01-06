import { Meta, StoryObj } from "@storybook/react";

import LastUpdatedHostCount from "./LastUpdatedHostCount";

const meta: Meta<typeof LastUpdatedHostCount> = {
  title: "Components/LastUpdatedHostCount",
  component: LastUpdatedHostCount,
  args: {
    // whatToRetrieve: "apples",
  },
};

export default meta;

type Story = StoryObj<typeof LastUpdatedHostCount>;

export const Basic: Story = {};

export const WithLastUpdatedAt: Story = {
  args: {
    lastUpdatedAt: "2021-01-01T00:00:00Z",
  },
};
