import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { ISoftware } from "interfaces/software";
import getMatchedSoftwareIcon, {
  SOFTWARE_NAME_TO_ICON_MAP,
  SOFTWARE_SOURCE_TO_ICON_MAP,
} from ".";

// Extend the props type to include the new selection prop
type IconWrapperProps = Pick<ISoftware, "name" | "source"> & {
  selection?: string;
};

const IconWrapper: React.FC<IconWrapperProps> = ({ selection, ...props }) => {
  const Icon = getMatchedSoftwareIcon(props);
  return <Icon />;
};

const meta: Meta<typeof IconWrapper> = {
  title: "Components/Icon/SoftwareIcon",
  component: IconWrapper,
  argTypes: {
    selection: {
      control: "select",
      options: [
        ...Object.keys(SOFTWARE_NAME_TO_ICON_MAP).map((name) => `name:${name}`),
        ...Object.keys(SOFTWARE_SOURCE_TO_ICON_MAP).map(
          (source) => `source:${source}`
        ),
      ],
    },
  },
};

export default meta;

type Story = StoryObj<typeof IconWrapper>;

export const Default: Story = {
  render: ({ selection }) => {
    if (!selection) return <IconWrapper name="" source="" />;

    const [type, value] = selection.split(":");
    const props =
      type === "name"
        ? { name: value, source: "" }
        : { name: "", source: value };
    return <IconWrapper {...props} />;
  },
};
