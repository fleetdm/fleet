import { Meta, StoryObj } from "@storybook/react";
import React from "react";

import { createMockConfig } from "__mocks__/configMock";
import Button from "components/buttons/Button";
import { AppContext, initialState } from "context/app";

import GitOpsModeTooltipWrapper from "./GitOpsModeTooltipWrapper";

/** Wraps stories in an AppContext with GitOps mode enabled so the wrapper's
 *  disabled+tooltip state renders. */
const withGitOpsMode = (Story: React.ComponentType) => (
  <AppContext.Provider
    value={{
      ...initialState,
      config: createMockConfig({
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-config",
          exceptions: { labels: false, software: false, secrets: true },
        },
      }),
    }}
  >
    <div style={{ padding: 24 }}>
      <Story />
    </div>
  </AppContext.Provider>
);

const withGitOpsDisabled = (Story: React.ComponentType) => (
  <AppContext.Provider value={{ ...initialState, config: createMockConfig() }}>
    <div style={{ padding: 24 }}>
      <Story />
    </div>
  </AppContext.Provider>
);

const meta: Meta<typeof GitOpsModeTooltipWrapper> = {
  component: GitOpsModeTooltipWrapper,
  title: "Components/GitOpsModeTooltipWrapper",
};

export default meta;

type Story = StoryObj<typeof GitOpsModeTooltipWrapper>;

export const GitOpsOff: Story = {
  name: "GitOps mode off (children pass through)",
  decorators: [withGitOpsDisabled],
  render: () => (
    <GitOpsModeTooltipWrapper
      renderChildren={(disabled) => <Button disabled={disabled}>Save</Button>}
    />
  ),
};

export const GitOpsOn: Story = {
  name: "GitOps mode on (button disabled + tooltip)",
  decorators: [withGitOpsMode],
  render: () => (
    <GitOpsModeTooltipWrapper
      renderChildren={(disabled) => <Button disabled={disabled}>Save</Button>}
    />
  ),
};

export const EntityExcepted: Story = {
  name: "GitOps on but entity excepted (children stay enabled)",
  decorators: [withGitOpsMode],
  render: () => (
    <GitOpsModeTooltipWrapper
      entityType="secrets"
      renderChildren={(disabled) => (
        <Button disabled={disabled}>Save enroll secret</Button>
      )}
    />
  ),
};

export const PositionLeft: Story = {
  name: "Left-positioned tooltip (partial-disable forms)",
  decorators: [withGitOpsMode],
  render: () => (
    <GitOpsModeTooltipWrapper
      position="left"
      renderChildren={(disabled) => <Button disabled={disabled}>Save</Button>}
    />
  ),
};
