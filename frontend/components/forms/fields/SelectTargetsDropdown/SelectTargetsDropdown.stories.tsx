import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import SelectedTargetsDropdown from ".";

import "../../../../index.scss";

const meta: Meta<typeof SelectedTargetsDropdown> = {
  component: SelectedTargetsDropdown,
  title: "Components/SelectTargetsDropdown",
  args: {
    disabled: false,
    label: "Select Targets",
    selectedTargets: [],
    targetsCount: 0,
    queryId: 1,
    onFetchTargets: noop,
    onSelect: noop,
  },
};

export default meta;

type Story = StoryObj<typeof SelectedTargetsDropdown>;

export const Default: Story = {};
