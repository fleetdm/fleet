import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import LastUpdatedText from "components/LastUpdatedText";

import SectionHeader from ".";

const meta: Meta<typeof SectionHeader> = {
  title: "Components/SectionHeader",
  component: SectionHeader,
  args: { title: "Section header title" },
};

export default meta;

type Story = StoryObj<typeof SectionHeader>;

export const Basic: Story = {};

export const WithSubTitle: Story = {
  args: {
    subTitle: (
      <LastUpdatedText
        lastUpdatedAt={new Date().toISOString()}
        whatToRetrieve="operating systems"
      />
    ),
  },
};
