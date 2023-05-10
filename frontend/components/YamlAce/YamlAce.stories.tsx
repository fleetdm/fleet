import { Meta, StoryObj } from "@storybook/react";
import { noop } from "lodash";

// @ts-ignore
import YamlAce from ".";

const TEST_YAML = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  containers:
    - name: my-container # comment
      image: nginx:1.14.2
      ports:
        - containerPort: 80
`;

const meta: Meta<typeof YamlAce> = {
  title: "Components/YamlAce",
  component: YamlAce,
  args: {
    value: TEST_YAML,
    onChange: noop,
  },
};

export default meta;

type Story = StoryObj<typeof YamlAce>;

export const Basic: Story = {};

export const WithError: Story = {
  args: {
    error: "This is an error",
  },
};
