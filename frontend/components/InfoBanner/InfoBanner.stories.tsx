import React from "react";
import { Meta, Story } from "@storybook/react";

// @ts-ignore
import InfoBanner from ".";
import { IInfoBannerProps } from "./InfoBanner";

import "../../index.scss";

export default {
  component: InfoBanner,
  title: "Components/InfoBanner",
} as Meta;

const Template: Story<IInfoBannerProps> = (props) => (
  <InfoBanner {...props}>
    <div>This is an Info Banner.</div>
  </InfoBanner>
);

export const Default = Template.bind({});
