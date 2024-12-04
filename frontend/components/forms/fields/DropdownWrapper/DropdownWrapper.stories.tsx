// stories/DropdownWrapper.stories.tsx

import React from "react";
import { Meta, Story } from "@storybook/react";
import DropdownWrapper, {
  IDropdownWrapper,
  CustomOptionType,
} from "./DropdownWrapper";

// Define metadata for the story
export default {
  title: "Components/DropdownWrapper",
  component: DropdownWrapper,
  argTypes: {
    onChange: { action: "changed" },
  },
} as Meta;

// Define a template for the stories
const Template: Story<IDropdownWrapper> = (args) => (
  <DropdownWrapper {...args} />
);

// Sample options to be used in the dropdown
const sampleOptions: CustomOptionType[] = [
  { label: "Option 1", value: "option1", helpText: "Help text for option 1" },
  {
    label: "Option 2",
    value: "option2",
    tooltipContent: "Tooltip for option 2",
  },
  { label: "Option 3", value: "option3", isDisabled: true },
];

// Default story
export const Default = Template.bind({});
Default.args = {
  options: sampleOptions,
  name: "dropdown-example",
  label: "Select an option",
};

// Disabled story
export const Disabled = Template.bind({});
Disabled.args = {
  ...Default.args,
  isDisabled: true,
};

// With Help Text story
export const WithHelpText = Template.bind({});
WithHelpText.args = {
  ...Default.args,
  helpText: "This is some help text for the dropdown",
};

// With Error story
export const WithError = Template.bind({});
WithError.args = {
  ...Default.args,
  error: "This is an error message",
};
