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

/** Hover "+N more" — the tooltip lists 10 items and counts the rest, rather
 * than growing past the height of the window. */
export const ManyItems: Story = {
  args: {
    items: Array.from({ length: 57 }, (_, i) => `CVE-2026-${1000 + i}`),
  },
  decorators: [withFrame(360)],
};
