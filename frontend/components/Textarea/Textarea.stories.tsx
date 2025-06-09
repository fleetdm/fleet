import { Meta, StoryObj } from "@storybook/react";
import Textarea from ".";
import "../../index.scss";

const meta: Meta<typeof Textarea> = {
  component: Textarea,
  title: "Components/Textarea",
  args: {
    children: "Default textarea content",
  },
};

export default meta;

type Story = StoryObj<typeof Textarea>;

export const ExampleWithLabelAndCodeVariant: Story = {
  args: {
    variant: "code",
    label: "Textarea label:",
    children:
      "const example = 'This is code styled text';\n// With line breaks preserved.\n\n Text with\r\ncarriage returns\r\nconverted to\nline breaks",
  },
};
