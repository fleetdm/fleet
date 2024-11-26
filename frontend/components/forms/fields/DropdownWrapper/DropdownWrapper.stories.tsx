import React from "react";
import { Meta, Story } from "@storybook/react";
import DropdownWrapper, { IDropdownWrapper } from "./DropdownWrapper";

export default {
  title: "Components/Forms/DropdownWrapper",
  component: DropdownWrapper,
  argTypes: {
    onChange: { action: "changed" },
  },
} as Meta;

const Template: Story<IDropdownWrapper> = (args) => (
  <DropdownWrapper {...args} />
);

export const Default = Template.bind({});
Default.args = {
  options: [
    { label: "Option 1", value: "observer", isDisabled: false },
    { label: "Option 2", value: "maintainer", isDisabled: false },
    { label: "Option 3", value: "admin", isDisabled: false },
  ],
  value: null,
  name: "default-dropdown",
  label: "Default Dropdown",
};

export const WithSearchable = Template.bind({});
WithSearchable.args = {
  ...Default.args,
  isSearchable: true,
  name: "searchable-dropdown",
  label: "Searchable Dropdown",
};

export const Disabled = Template.bind({});
Disabled.args = {
  ...Default.args,
  isDisabled: true,
  name: "disabled-dropdown",
  label: "Disabled Dropdown",
};

export const WithError = Template.bind({});
WithError.args = {
  ...Default.args,
  error: "This is an error message",
  name: "error-dropdown",
  label: "Dropdown with Error",
};

export const WithTooltip = Template.bind({});
WithTooltip.args = {
  options: [
    {
      label: "Option 1",
      value: "Observer",
      helpText: "This is help text for Option 1",
      isDisabled: false,
    },
    {
      label: "Option 2",
      value: "Maintainer",
      helpText: "This is help text for Option 2",
      isDisabled: false,
    },
    {
      label: "Option 3",
      value: "Admin",
      helpText: "This is help text for Option 3",
      isDisabled: false,
    },
  ],
  value: null,
  name: "tooltip-dropdown",
  label: "Dropdown with Tooltips",
};

export const WithHelpText = Template.bind({});
WithHelpText.args = {
  options: [
    {
      label: "Option 1",
      value: "observer",
      helpText: "This is help text for Option 1",
      isDisabled: false,
    },
    {
      label: "Option 2",
      value: "maintainer",
      helpText: "This is help text for Option 2",
      isDisabled: false,
    },
    {
      label: "Option 3",
      value: "admin",
      helpText: "This is help text for Option 3",
      isDisabled: false,
    },
  ],
  value: null,
  name: "helptext-dropdown",
  label: "Dropdown with Help Text",
};
