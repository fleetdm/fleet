import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import Icon from "components/Icon";
import { ButtonVariant } from "./Button";
import Button from ".";

const DEFAULT_ARGS = {
  children: "Button text",
  onClick: () => console.log("Clicked!"),
  disabled: false,
};

const meta: Meta<typeof Button> = {
  // TODO: change this after button is updated to a functional component. For
  // some reason the typing is incorrect because Button is a class component.
  component: Button as any,
  title: "Components/Button",
  argTypes: {
    variant: { control: false }, // Hide variant control since we're making separate stories
    disabled: {
      control: { type: "boolean" }, // Make disabled an easy toggle switch
      table: {
        defaultValue: { summary: "false" }, // Show default value in docs
        description: "Disabled state Viewable in Figma*", // Figma link indicator
      },
    },
  },
  args: DEFAULT_ARGS,
};

export default meta;
type Story = StoryObj<typeof Button>;

// Base template for NON-loading variants (explicitly hides isLoading)
const Template = (variant: ButtonVariant, children?: JSX.Element): Story => ({
  args: {
    ...DEFAULT_ARGS,
    variant,
    children: children || DEFAULT_ARGS.children, // Fall back to default text
  },
  argTypes: {
    isLoading: { control: false }, // Explicitly hide for these
  },
});

// Template for loading-enabled variants
const createLoadingVariant = (variant: ButtonVariant): Story => ({
  args: {
    ...DEFAULT_ARGS,
    variant,
    isLoading: false, // Include isLoading in args
  },
  argTypes: {
    isLoading: {
      control: { type: "boolean" },
      table: {
        defaultValue: { summary: "false" },
        description: "Shows loading spinner (only be used on these buttons)",
      },
    },
  },
});

// Variants with loading state
export const DefaultVariant = createLoadingVariant("default");
export const SuccessVariant = createLoadingVariant("success");
export const AlertVariant = createLoadingVariant("alert");
export const InverseVariant = Template("inverse");
export const InverseAlertVariant = Template("inverse-alert");

export const PillVariant = Template("pill");
export const TextLinkVariant = Template("text-link");
export const TextLinkDarkVariant = Template("text-link-dark");
export const TextIconVariant = Template(
  "text-icon",
  <>
    Button text <Icon name="plus" size="small" />
  </>
);
export const IconVariant = Template("text-icon", <Icon name="trash" />);

export const UnstyledVariant = Template("unstyled");
export const UnstyledModalQueryVariant = Template("unstyled-modal-query");
export const OversizedVariant = Template("oversized");
