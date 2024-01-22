import { Meta, StoryObj } from "@storybook/react";

import { HumanTimeDiffWithDateTip } from "./HumanTimeDiffWithDateTip";

const meta: Meta<typeof HumanTimeDiffWithDateTip> = {
  title: "Components/HumanTimeDiffWithDateTip",
  component: HumanTimeDiffWithDateTip,
  args: {
    timeString: "2021-01-01T00:00:00.000Z",
  },
};

export default meta;

type Story = StoryObj<typeof HumanTimeDiffWithDateTip>;

export const Basic: Story = {};
