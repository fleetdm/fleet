import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import CustomLink from "components/CustomLink";
import InfoBanner from ".";
import "../../index.scss";

const meta: Meta<typeof InfoBanner> = {
  component: InfoBanner,
  title: "Components/InfoBanner",
  argTypes: {
    color: {
      control: { type: "select" },
      options: ["purple", "yellow", "grey"],
    },
    borderRadius: {
      control: { type: "select" },
      options: ["medium", "xlarge"],
    },
    pageLevel: { control: "boolean" },
    closable: { control: "boolean" },
    icon: {
      control: { type: "select" },
      options: [undefined, "info", "warning", "error", "success"],
    },
    cta: { table: { disable: true } }, // Kept as is for JSX
    children: { table: { disable: true }, description: "React.ReactNode" },
    className: { control: "text" },
  },
  parameters: { controls: { expanded: true } },
};

export default meta;

type Story = StoryObj<typeof InfoBanner>;

const defaultChildren = (
  <>
    <b>Fleet</b> is unable to run a live query. Refresh the page or log in
    again. If this keeps happening please{" "}
    <CustomLink
      url="https://github.com/fleetdm/fleet/issues/new/choose"
      text="file an issue"
      newTab
      variant="banner-link"
    />
  </>
);

const sampleCta = (
  <CustomLink
    url="http://localhost:6006/"
    text="Reopen Storybook"
    newTab
    variant="banner-link"
  />
);

export const Playground: Story = {
  args: {
    children: defaultChildren,
    cta: sampleCta,
    color: "purple",
    borderRadius: "medium",
    pageLevel: false,
    closable: true,
    icon: "info",
  },
  render: (args) => <InfoBanner {...args} />,
};
