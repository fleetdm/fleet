import { Meta, StoryObj } from "@storybook/react";

import Chip from "./Chip";
import "../../index.scss";

const meta: Meta<typeof Chip> = {
  component: Chip,
  title: "Components/Chip",
  argTypes: {
    icon: { control: "text" },
    trailingIcon: { control: "text" },
    text: { control: "text" },
    tooltip: { control: "text" },
    onClick: { table: { disable: true } },
    className: { control: "text" },
  },
  parameters: { controls: { expanded: true } },
};

export default meta;

type Story = StoryObj<typeof Chip>;

export const Playground: Story = {
  args: {
    text: "Fleet-maintained",
  },
};

export const WithLeadingIcon: Story = {
  args: {
    icon: "user",
    text: "Self-service",
  },
};

export const WithTrailingIcon: Story = {
  args: {
    icon: "refresh",
    text: "Auto install",
    trailingIcon: "chevron-right",
  },
};

export const Clickable: Story = {
  args: {
    icon: "refresh",
    text: "Auto install",
    onClick: () => undefined,
  },
};

export const WithTooltip: Story = {
  args: {
    icon: "user",
    text: "Self-service",
    tooltip: "End users can install this from the My device page.",
  },
};

export const ClickableWithTooltip: Story = {
  args: {
    icon: "refresh",
    text: "Auto install",
    onClick: () => undefined,
    tooltip: "Policy triggers install.",
  },
};
