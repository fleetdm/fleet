import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";
import React, { useState } from "react";

import { ILabelSummary } from "interfaces/label";

import TargetLabelSelector, {
  ILabelConfig,
  LabelTargetMode,
  TargetType,
} from "./TargetLabelSelector";

const LABELS: ILabelSummary[] = [
  { id: 1, name: "Engineering laptops", label_type: "regular" },
  { id: 2, name: "Executive team", label_type: "regular" },
  { id: 3, name: "Kiosks", label_type: "regular" },
  { id: 4, name: "Contractors", label_type: "regular" },
  { id: 5, name: "macOS", label_type: "builtin" },
  { id: 6, name: "MS Windows", label_type: "builtin" },
];

const meta: Meta<typeof TargetLabelSelector> = {
  component: TargetLabelSelector,
  // Sibling DropdownTargetLabelSelector is documented separately; this one is
  // the tabbed Include/Exclude experience used by policies and config profiles.
  title: "Components/TargetLabelSelector/Tabbed",
  // Stacking every preview on one Docs page merges all target-type radios
  // into one radio group (the component hardcodes name="target-type" and id
  // on each radio); opt out so each story renders standalone.
  tags: ["!autodocs"],
  args: {
    labels: LABELS,
    emptyStateDescription:
      "Add a label so you can target a subset of hosts here.",
    onAddLabel: noop,
  },
  decorators: [
    (Story) => (
      <div style={{ maxWidth: 560 }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof TargetLabelSelector>;

/** Renders TargetLabelSelector wired to local state so the include / exclude
 *  tabs behave the way they do in-product. */
const Interactive = ({
  initialTargetType = "Custom",
  initialInclude = {},
  initialExclude = {},
  initialIncludeMode = "any",
  showIncludeToggle = true,
  showExcludeToggle = false,
  ...meta_args
}: {
  initialTargetType?: TargetType;
  initialInclude?: Record<string, boolean>;
  initialExclude?: Record<string, boolean>;
  initialIncludeMode?: LabelTargetMode;
  showIncludeToggle?: boolean;
  showExcludeToggle?: boolean;
} & Partial<React.ComponentProps<typeof TargetLabelSelector>>) => {
  const [targetType, setTargetType] = useState<TargetType>(initialTargetType);
  const [include, setInclude] = useState<Record<string, boolean>>(
    initialInclude
  );
  const [exclude, setExclude] = useState<Record<string, boolean>>(
    initialExclude
  );
  const [includeMode, setIncludeMode] = useState<LabelTargetMode>(
    initialIncludeMode
  );

  const includeConfig: ILabelConfig = {
    selectedLabels: include,
    onSelectLabel: ({ name, value }) =>
      setInclude((prev) => ({ ...prev, [name]: value })),
    showModeToggle: showIncludeToggle,
    mode: includeMode,
    onSelectMode: setIncludeMode,
  };

  const excludeConfig: ILabelConfig = {
    selectedLabels: exclude,
    onSelectLabel: ({ name, value }) =>
      setExclude((prev) => ({ ...prev, [name]: value })),
    showModeToggle: showExcludeToggle,
  };

  return (
    <TargetLabelSelector
      {...meta_args}
      labels={meta_args.labels ?? LABELS}
      emptyStateDescription={meta_args.emptyStateDescription ?? "Add a label."}
      onAddLabel={meta_args.onAddLabel ?? noop}
      selectedTargetType={targetType}
      onSelectTargetType={setTargetType}
      includeConfig={includeConfig}
      excludeConfig={excludeConfig}
    />
  );
};

export const AllHosts: Story = {
  render: () => <Interactive initialTargetType="All hosts" />,
};

export const CustomEmpty: Story = {
  render: () => <Interactive />,
};

export const WithPreselectedLabels: Story = {
  render: () => (
    <Interactive
      initialInclude={{ "Engineering laptops": true }}
      initialExclude={{ Contractors: true }}
    />
  ),
};

export const IncludeModeAll: Story = {
  name: "Include mode = All (labels_include_all)",
  render: () => <Interactive initialIncludeMode="all" />,
};

export const Loading: Story = {
  render: () => <Interactive isLoadingLabels />,
};

export const Error: Story = {
  render: () => <Interactive isErrorLabels />,
};

export const NoLabelsEmptyState: Story = {
  render: () => <Interactive labels={[]} />,
};

export const Disabled: Story = {
  render: () => (
    <Interactive
      disableOptions
      initialInclude={{ "Engineering laptops": true }}
    />
  ),
};
