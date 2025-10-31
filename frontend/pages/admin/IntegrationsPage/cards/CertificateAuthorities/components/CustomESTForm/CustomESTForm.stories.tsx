import { Meta, StoryObj } from "@storybook/react";

import CustomESTForm from "./CustomESTForm";

const meta: Meta<typeof CustomESTForm> = {
  title: "Components/CustomESTForm",
  component: CustomESTForm,
};

export default meta;

type Story = StoryObj<typeof CustomESTForm>;

export const Basic: Story = {};
