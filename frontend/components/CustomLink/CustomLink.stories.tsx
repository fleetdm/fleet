import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from ".";

const meta: Meta<typeof CustomLink> = {
  title: "Components/CustomLink",
  component: CustomLink,
};

export default meta;

type Story = StoryObj<typeof CustomLink>;

export const Basic: Story = {
  args: {
    url: "https://www.google.com",
    text: "Test Link",
  },
};

export const ExternalLink: Story = {
  args: {
    ...Basic.args,
    newTab: true,
  },
};

export const Multiline: Story = {
  render: (args) => (
    <div
      style={{
        width: "400px",
      }}
    >
      Here&apos;s a CustomLink in a that might be split up across two lines{" "}
      <CustomLink {...args} />
    </div>
  ),
  args: {
    url: "https://www.google.com",
    text:
      "This is a custom link that has multiple words that might span multiple lines and the icon should stick with the last word onto the new line",
    multiline: true,
    newTab: true,
  },
};

export const TooltipVariant: Story = {
  render: (args) => (
    <TooltipWrapper
      tipContent={
        <>
          Tip content with a custom link <CustomLink {...args} />
        </>
      }
    >
      Hover to see custom link in tooltip{" "}
    </TooltipWrapper>
  ),
  args: {
    url: "https://www.google.com",
    text: "Tooltip link",
    variant: "tooltip-link",
    newTab: true,
  },
};

export const BannerVariant: Story = {
  render: (args) => (
    <InfoBanner>
      Here&apos;s a CustomLink in a banner <CustomLink {...args} />
    </InfoBanner>
  ),
  args: {
    url: "https://www.google.com",
    text: "Banner link",
    variant: "banner-link",
    newTab: true,
  },
};

export const FlashMessageVariant: Story = {
  args: {
    url: "https://www.google.com",
    text: "Flash message link",
    variant: "flash-message-link",
  },
};

export const DisabledKeyboardNav: Story = {
  render: (args) => (
    <>
      Here, you can&apos;t tab to this link even if you wanted to which is
      useful when within a disabled component. <CustomLink {...args} />
    </>
  ),
  args: {
    url: "https://www.google.com",
    text: "Disabled Keyboard Navigation",
    disableKeyboardNavigation: true,
  },
};
