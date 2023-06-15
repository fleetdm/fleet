import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

import PlatformSelector from "./PlatformSelector";

const meta: Meta<typeof PlatformSelector> = {
  title: "Components/PlatformSelector",
  component: PlatformSelector,
  args: {
    checkDarwin: true,
    checkWindows: true,
    checkLinux: false,
    checkChrome: false,
    setCheckDarwin: noop,
    setCheckWindows: noop,
    setCheckLinux: noop,
    setCheckChrome: noop,
  },
};

export default meta;

type Story = StoryObj<typeof PlatformSelector>;

export const Basic: Story = {};
