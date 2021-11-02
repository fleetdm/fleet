import React from "react";
import { Meta, Story } from "@storybook/react";
import { noop } from "lodash";

import { ITarget } from "interfaces/target"; // @ts-ignore
import SelectedTargetsDropdown from ".";

import "../../../../index.scss";

interface ISelectedTargetsDropdownProps {
  disabled?: boolean;
  error?: string;
  label?: string;
  selectedTargets?: ITarget[];
  targetsCount?: number;
  queryId?: number;
  isPremiumTier?: boolean;
  onSelect: () => void;
  onFetchTargets?: () => void;
}

export default {
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
} as Meta;

const Template: Story<ISelectedTargetsDropdownProps> = (props) => (
  <SelectedTargetsDropdown {...props} />
);

export const Default = Template.bind({});
