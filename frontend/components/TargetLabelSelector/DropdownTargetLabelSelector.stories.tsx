import { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";

import { ILabelSummary } from "interfaces/label";

import DropdownTargetLabelSelector from "./DropdownTargetLabelSelector";
import { getCustomTargetOptions } from "./labelScopes";

const LABELS: ILabelSummary[] = [
  { id: 1, name: "Engineering laptops", label_type: "regular" },
  { id: 2, name: "Executive team", label_type: "regular" },
  { id: 3, name: "Kiosks", label_type: "regular" },
  { id: 4, name: "Contractors", label_type: "regular" },
  { id: 5, name: "macOS", label_type: "builtin" },
  { id: 6, name: "MS Windows", label_type: "builtin" },
];

const REPORT_OPTIONS = getCustomTargetOptions({
  entity: "report",
  isPremiumTier: true,
});
const POLICY_OPTIONS = getCustomTargetOptions({
  entity: "policy",
  isPremiumTier: true,
});
const REPORT_OPTIONS_FREE = getCustomTargetOptions({
  entity: "report",
  isPremiumTier: false,
});

interface IInteractiveProps
  extends Partial<React.ComponentProps<typeof DropdownTargetLabelSelector>> {
  initialTargetType?: string;
  initialCustomTarget?: string;
  initialSelectedLabels?: Record<string, boolean>;
}

const Interactive = ({
  initialTargetType = "Custom",
  initialCustomTarget = "labelsIncludeAny",
  initialSelectedLabels = {},
  customTargetOptions = REPORT_OPTIONS,
  labels = LABELS,
  ...rest
}: IInteractiveProps) => {
  const [targetType, setTargetType] = useState(initialTargetType);
  const [customTarget, setCustomTarget] = useState(initialCustomTarget);
  const [selectedLabels, setSelectedLabels] = useState<Record<string, boolean>>(
    initialSelectedLabels
  );

  return (
    // Spread `rest` first so state-backed selection callbacks below can't be
    // clobbered by Storybook args (e.g. auto-generated action handlers).
    <DropdownTargetLabelSelector
      {...rest}
      customTargetOptions={customTargetOptions}
      labels={labels}
      selectedTargetType={targetType}
      onSelectTargetType={setTargetType}
      selectedCustomTarget={customTarget}
      onSelectCustomTarget={setCustomTarget}
      selectedLabels={selectedLabels}
      onSelectLabel={({ name, value }) =>
        setSelectedLabels((prev) => ({ ...prev, [name]: value }))
      }
    />
  );
};

const meta: Meta<typeof DropdownTargetLabelSelector> = {
  component: DropdownTargetLabelSelector,
  title: "Components/TargetLabelSelector/Dropdown",
  // Stacking every preview on one Docs page merges all target-type radios
  // into one radio group (the component hardcodes name="target-type" and id
  // on each radio); opt out so each story renders standalone.
  tags: ["!autodocs"],
  decorators: [
    (Story) => (
      <div style={{ maxWidth: 560 }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof DropdownTargetLabelSelector>;

export const AllHosts: Story = {
  render: (args) => <Interactive {...args} initialTargetType="All hosts" />,
};

export const ReportCustom: Story = {
  name: "Report (premium): Include any, Include all",
  render: (args) => (
    <Interactive {...args} customTargetOptions={REPORT_OPTIONS} />
  ),
};

export const PolicyCustom: Story = {
  name: "Policy (premium): Include any, Include all, Exclude any",
  render: (args) => (
    <Interactive {...args} customTargetOptions={POLICY_OPTIONS} />
  ),
};

export const ReportCustomFree: Story = {
  name: "Report (Free): Include any only",
  render: (args) => (
    <Interactive {...args} customTargetOptions={REPORT_OPTIONS_FREE} />
  ),
};

export const WithPreselectedLabels: Story = {
  render: (args) => (
    <Interactive
      {...args}
      initialSelectedLabels={{
        "Engineering laptops": true,
        "Executive team": true,
      }}
    />
  ),
};

export const OverrideDropdownHelpText: Story = {
  name: "Override dropdown help text",
  render: (args) => (
    <Interactive
      {...args}
      dropdownHelpText="Custom help text overrides the per-option copy."
    />
  ),
};

export const SuppressTitle: Story = {
  name: "Suppress title (embedded in modal)",
  render: (args) => <Interactive {...args} suppressTitle />,
};

export const WithSubtitle: Story = {
  render: (args) => (
    <Interactive
      {...args}
      title="Target"
      subTitle="Report runs on hosts matching these labels."
    />
  ),
};

export const Loading: Story = {
  render: (args) => <Interactive {...args} isLoadingLabels />,
};

export const Error: Story = {
  render: (args) => <Interactive {...args} isErrorLabels />,
};

export const NoLabels: Story = {
  render: (args) => <Interactive {...args} labels={[]} />,
};

export const Disabled: Story = {
  render: (args) => (
    <Interactive
      {...args}
      disableOptions
      initialSelectedLabels={{ "Engineering laptops": true }}
    />
  ),
};
