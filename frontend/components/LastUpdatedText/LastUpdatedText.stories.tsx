import { Meta, StoryObj } from "@storybook/react";

import LastUpdatedText from "./LastUpdatedText";

const meta: Meta<typeof LastUpdatedText> = {
  title: "Components/LastUpdatedText",
  component: LastUpdatedText,
  args: {
    whatToRetrieve: "apples",
  },
};

export default meta;

type Story = StoryObj<typeof LastUpdatedText>;

export const Basic: Story = {};

export const WithLastUpdatedAt: Story = {
  args: {
    lastUpdatedAt: "2021-01-01T00:00:00Z",
  },
};
