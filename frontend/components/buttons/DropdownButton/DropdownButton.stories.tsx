import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";
import React from "react";

import { createMockConfig } from "__mocks__/configMock";
import Avatar from "components/Avatar";
import { AppContext, initialState } from "context/app";
import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

// @ts-ignore
import DropdownButton from ".";

import "../../../index.scss";

const options = [
  {
    label: "My account",
    onClick: noop,
  },
  {
    label: "Documentation",
    onClick: () => window.open("https://fleetdm.com/docs", "_blank"),
  },
  {
    label: "Sign out",
    onClick: noop,
  },
];

// Wraps stories in an AppContext with GitOps mode enabled so disabled
// GitOps-compatible options render their tooltip.
const withGitOpsMode = (Story: React.ComponentType) => (
  <AppContext.Provider
    value={{
      ...initialState,
      config: createMockConfig({
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-config",
        },
      }),
    }}
  >
    <div style={{ padding: 24 }}>
      <Story />
    </div>
  </AppContext.Provider>
);

const meta: Meta<typeof DropdownButton> = {
  component: DropdownButton,
  title: "Components/DropdownButton",
  argTypes: {
    variant: {
      options: [
        "default",
        "alert",
        "pill",
        "link",
        "unstyled",
        "unstyled-modal-query",
      ],
      control: "select",
    },
    type: {
      options: ["button", "submit", "reset"],
      control: "select",
    },
  },
  args: {
    variant: "unstyled",
    className: "story",
    size: "",
    tabIndex: 0,
    options,
  },
};

export default meta;

type Story = StoryObj<typeof DropdownButton>;

export const Default: Story = {
  args: {
    children: (
      <Avatar user={{ gravatar_url: DEFAULT_GRAVATAR_LINK }} size="small" />
    ),
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    disabled: true,
  },
};

export const GitOpsDisabledItems: Story = {
  name: "GitOps mode (disabled GitOps-compatible items show tooltip)",
  decorators: [withGitOpsMode],
  args: {
    children: "More options",
    options: [
      {
        label: "Rename fleet",
        onClick: noop,
        disabled: true,
        gitOpsModeCompatible: true,
      },
      {
        label: "Delete fleet",
        onClick: noop,
        disabled: true,
        gitOpsModeCompatible: true,
      },
    ],
  },
};
