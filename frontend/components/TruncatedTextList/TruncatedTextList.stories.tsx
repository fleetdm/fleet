import { Meta, StoryObj } from "@storybook/react";

import withFrame from "test/storybook-utils";

import TruncatedTextList from "./TruncatedTextList";

const meta: Meta<typeof TruncatedTextList> = {
  title: "Components/TruncatedTextList",
  component: TruncatedTextList,
  args: {
    items: [
      "Engineering",
      "Product",
      "Quality Assurance",
      "Marketing",
      "Sales",
      "Support",
      "Operations",
    ],
  },
};

export default meta;

type Story = StoryObj<typeof TruncatedTextList>;

export const Basic: Story = {
  decorators: [withFrame(360)],
};

export const NarrowContainer: Story = {
  args: { truncatedFirstMaxChars: 6 },
  decorators: [withFrame(180)],
};

export const AllFit: Story = {
  args: { items: ["Mac", "Linux"] },
  decorators: [withFrame(360)],
};
