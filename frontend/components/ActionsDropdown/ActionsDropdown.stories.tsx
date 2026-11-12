import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";
import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

import ActionsDropdown from "./ActionsDropdown";

const BASIC_OPTIONS: IDropdownOption[] = [
  { label: "Edit", value: "edit" },
  { label: "Duplicate", value: "duplicate" },
  { label: "Delete", value: "delete" },
];

const RICH_OPTIONS: IDropdownOption[] = [
  {
    label: "Run script",
    value: "run",
    helpText: "Executes on this host now.",
  },
  {
    label: "Lock",
    value: "lock",
    helpText: "Requires an unlock PIN.",
  },
  { label: "Wipe", value: "wipe", helpText: "Irreversible." },
  {
    label: "Transfer",
    value: "transfer",
    disabled: true,
    tooltipContent: "You need admin on both fleets to move this host.",
  },
];

const meta: Meta<typeof ActionsDropdown> = {
  component: ActionsDropdown,
  title: "Components/ActionsDropdown",
  argTypes: {
    variant: {
      control: { type: "select" },
      options: ["primary", "secondary", "subdued"],
    },
    menuAlign: {
      control: { type: "select" },
      options: ["default", "left", "right"],
    },
    menuPlacement: {
      control: { type: "select" },
      options: [undefined, "top", "bottom", "auto"],
    },
    disabled: { control: "boolean" },
  },
  args: {
    options: BASIC_OPTIONS,
    placeholder: "Actions",
    onChange: noop,
  },
  // Enough vertical room for the menu to expand downward inside the frame.
  decorators: [
    (Story) => (
      <div style={{ minHeight: 260, padding: 24 }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof ActionsDropdown>;

export const Subdued: Story = {
  args: { variant: "subdued" },
};

export const Secondary: Story = {
  args: { variant: "secondary" },
};

export const Primary: Story = {
  // In-product the primary variant is always paired with menuAlign="right"
  // (e.g. HostActionsDropdown); that pairing is what pins the menu to the
  // trigger. With the default alignment the nulled Control collapses to 0px
  // and the menu drops from an empty slot beside the button.
  args: { variant: "primary", buttonLabel: "Actions", menuAlign: "right" },
};

export const IconTrigger: Story = {
  args: { triggerIcon: "more", placeholder: "Row actions" },
};

export const WithHelpTextAndDisabledOption: Story = {
  args: { options: RICH_OPTIONS, variant: "subdued" },
};

export const Disabled: Story = {
  args: { disabled: true },
};

export const MenuAlignRight: Story = {
  name: "Menu aligned to right of trigger",
  args: { menuAlign: "right" },
  render: (args) => (
    <div style={{ display: "flex", justifyContent: "flex-end" }}>
      <ActionsDropdown {...args} />
    </div>
  ),
};
