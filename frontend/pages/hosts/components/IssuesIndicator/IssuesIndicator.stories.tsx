import { Meta, StoryObj } from "@storybook/react";

import IssuesIndicator from "./IssuesIndicator";

const meta: Meta<typeof IssuesIndicator> = {
  title: "Components/IssuesIndicator",
  component: IssuesIndicator,
  args: {
    totalIssuesCount: 5,
    criticalVulnerabilitiesCount: 3,
    failingPoliciesCount: 2,
  },
};

export default meta;

type Story = StoryObj<typeof IssuesIndicator>;

export const Basic: Story = {};
