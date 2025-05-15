import React from "react";
import InputField from ".";

export default {
  component: InputField,
  title: "Components/FormFields/InputField",
  argTypes: {
    type: {
      control: "select",
      options: ["text", "password", "email", "number", "textarea"],
    },
    value: {
      control: "text",
    },
    placeholder: {
      control: "text",
    },
    label: {
      control: "text",
    },
    error: {
      control: "text",
    },
    helpText: {
      control: "text",
    },
    disabled: {
      control: "boolean",
    },
    readOnly: {
      control: "boolean",
    },
    autofocus: {
      control: "boolean",
    },
    enableCopy: {
      control: "boolean",
    },
    enableShowSecret: {
      control: "boolean",
    },
  },
};

const Template = (args) => <InputField {...args} />;

export const Basic = Template.bind({});
Basic.args = {
  name: "basic-input",
  label: "Basic Input",
  value: "",
  placeholder: "Enter text here",
};

export const WithValue = Template.bind({});
WithValue.args = {
  ...Basic.args,
  value: "Sample text",
};

export const WithError = Template.bind({});
WithError.args = {
  ...Basic.args,
  error: "This field is required",
};

export const Disabled = Template.bind({});
Disabled.args = {
  ...Basic.args,
  disabled: true,
};

export const ReadOnly = Template.bind({});
ReadOnly.args = {
  ...Basic.args,
  readOnly: true,
  value: "Read-only content",
};

export const WithHelpText = Template.bind({});
WithHelpText.args = {
  ...Basic.args,
  helpText: "This is some helpful information about the input field.",
};

export const Password = Template.bind({});
Password.args = {
  ...Basic.args,
  type: "password",
  label: "Password",
  placeholder: "Enter your password",
};

export const Textarea = Template.bind({});
Textarea.args = {
  ...Basic.args,
  type: "textarea",
  label: "Text area",
  placeholder: "Enter multiple lines of text",
};

export const WithCopyEnabled = Template.bind({});
WithCopyEnabled.args = {
  ...Basic.args,
  enableCopy: true,
  value: "This text can be copied",
};

export const WithCopyEnabledInput = Template.bind({});
WithCopyEnabledInput.args = {
  ...WithCopyEnabled.args,
};

export const WithTooltip = Template.bind({});
WithTooltip.args = {
  ...Basic.args,
  tooltip: "This is a tooltip for the input field",
};

export const AutoFocus = Template.bind({});
AutoFocus.args = {
  ...Basic.args,
  autofocus: true,
};
