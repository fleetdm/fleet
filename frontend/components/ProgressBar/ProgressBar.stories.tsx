import { Meta, StoryObj } from "@storybook/react";

import ProgressBar from "./ProgressBar";

const meta: Meta<typeof ProgressBar> = {
  title: "Components/ProgressBar",
  component: ProgressBar,
};

export default meta;

type Story = StoryObj<typeof ProgressBar>;

export const Basic: Story = {
  args: {
    sections: [
      { color: "#5cb85c", portion: 0.85 },
      { color: "#d9534f", portion: 0.05 },
    ],
  },
};

export const MultipleSegments: Story = {
  args: {
    sections: [
      { color: "#5cb85c", portion: 0.3 },
      { color: "#f0ad4e", portion: 0.3 },
      { color: "#d9534f", portion: 0.2 },
      { color: "#0275d8", portion: 0.1 },
    ],
  },
};

export const SingleSegment: Story = {
  args: {
    sections: [{ color: "#5cb85c", portion: 0.5 }],
  },
};

export const CustomBackground: Story = {
  args: {
    sections: [{ color: "#5cb85c", portion: 0.6 }],
    backgroundColor: "#f8d7da", // Light red background
  },
};

export const DefaultBackground: Story = {
  args: {
    sections: [{ color: "#0275d8", portion: 0.3 }],
    // Uses default grey background
  },
};
