import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import Icon from "components/Icon";
import { ButtonVariant, IButtonProps } from "./Button";
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
const Template = (
  variant: ButtonVariant,
  children?: JSX.Element,
  extraArgs?: Partial<IButtonProps> // e.g. { size: "small" } or { disabled: true }
): Story => ({
  args: {
    ...DEFAULT_ARGS,
    variant,
    children: children || DEFAULT_ARGS.children, // Fall back to default text
    ...extraArgs,
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
// Used for Action dropdown triggers in the product.
export const DefaultIconAfterVariant = Template("default", undefined, {
  rightIcon: "chevron-right",
});

export const AlertVariant = createLoadingVariant("alert");

// Bordered secondary button — see #35329
export const SecondaryVariant = Template("secondary");
export const SecondaryIconAfterVariant = Template("secondary", undefined, {
  rightIcon: "plus",
});
export const SecondaryIconBeforeVariant = Template("secondary", undefined, {
  leftIcon: "plus",
});
export const SecondaryIconOnlyVariant = Template(
  "secondary",
  <Icon name="trash" />
);
export const SecondarySmallVariant = Template("secondary", undefined, {
  size: "small",
});
export const SecondarySmallIconBeforeVariant = Template(
  "secondary",
  undefined,
  {
    size: "small",
    leftIcon: "plus",
  }
);
export const SecondaryDisabledVariant = Template("secondary", undefined, {
  disabled: true,
});

// Borderless subdued button (low-emphasis text + icon)
export const SubduedIconAfterVariant = Template("subdued", undefined, {
  rightIcon: "chevron-right",
});
export const SubduedIconBeforeVariant = Template("subdued", undefined, {
  leftIcon: "chevron-left",
});
export const SubduedIconOnlyVariant = Template(
  "subdued",
  <Icon name="chevron-right" />
);
export const SubduedSmallVariant = Template("subdued", undefined, {
  size: "small",
  rightIcon: "chevron-right",
});
export const SubduedDisabledVariant = Template("subdued", undefined, {
  rightIcon: "chevron-right",
  disabled: true,
});

export const PillVariant = Template("pill");
export const LinkVariant = Template("link");

export const UnstyledVariant = Template("unstyled");
export const UnstyledModalQueryVariant = Template("unstyled-modal-query");
export const OversizedVariant = Template("oversized");
