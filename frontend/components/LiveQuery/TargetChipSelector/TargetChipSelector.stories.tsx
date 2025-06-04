import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { ISelectLabel, ISelectTeam } from "interfaces/target";
import TargetChipSelector from "./TargetChipSelector"; // Adjust the path if necessary

const meta: Meta<typeof TargetChipSelector> = {
  component: TargetChipSelector,
  title: "Components/TargetChipSelector",
  argTypes: {
    entity: {
      description: "The label or team entity to display.",
      control: { type: "object" },
    },
    isSelected: {
      description:
        "Whether the chip is currently selected, updated by parent onClick handler.",
      control: { type: "boolean" },
    },
    onClick: {
      description: "The handler to call when the chip is clicked.",
      action: "clicked", // Use Storybook's action to track clicks
    },
  },
  parameters: {
    backgrounds: {
      default: "light",
      values: [
        { name: "light", value: "#ffffff" },
        { name: "dark", value: "#333333" },
      ],
    },
  },
};

export default meta;

type Story = StoryObj<typeof TargetChipSelector>;

// Example data for labels and teams
const mockLabel: ISelectLabel = {
  id: 1,
  name: "Example Label",
  label_type: "regular",
  description: "A test label",
};

const mockTeam: ISelectTeam = {
  id: 2,
  name: "Example Team",
  description: "A test team",
};

export const LabelExample: Story = {
  args: {
    entity: mockLabel,
    isSelected: false,
    onClick: (value) => (event) => {
      event.preventDefault();
      console.log("Clicked label:", value);
    },
  },
  render: (args) => (
    <TargetChipSelector
      entity={args.entity}
      isSelected={args.isSelected}
      onClick={args.onClick}
    />
  ),
};

export const TeamExample: Story = {
  args: {
    entity: mockTeam,
    isSelected: true,
    onClick: (value) => (event) => {
      event.preventDefault();
      console.log("Clicked team:", value);
    },
  },
  render: (args) => (
    <TargetChipSelector
      entity={args.entity}
      isSelected={args.isSelected}
      onClick={args.onClick}
    />
  ),
};

export const BuiltInLabelExample: Story = {
  args: {
    entity: {
      id: 3,
      name: "MS Windows",
      label_type: "builtin",
      description: "Microsoft Windows hosts",
    },
    isSelected: false,
    onClick: (value) => (event) => {
      event.preventDefault();
      console.log("Clicked label:", value);
    },
  },
  render: (args) => (
    <TargetChipSelector
      entity={args.entity}
      isSelected={args.isSelected}
      onClick={args.onClick}
    />
  ),
};
