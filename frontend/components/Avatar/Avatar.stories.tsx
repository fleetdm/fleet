import React from "react";
import { Meta } from "@storybook/react";

import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

import Avatar from ".";

// import "./_styles.scss";
import "../../index.scss";

export default {
  component: Avatar,
  title: "Components/Avatar",
} as Meta;

export const Default = () => (
  <Avatar user={{ gravatarURL: DEFAULT_GRAVATAR_LINK }} />
);
export const Small = () => (
  <Avatar user={{ gravatarURL: DEFAULT_GRAVATAR_LINK }} size="small" />
);
