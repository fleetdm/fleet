import InputField from ".";

const meta = {
  component: InputField,
  title: "Components/FormFields/InputField",
};

export default meta;

export const Basic = {};

export const WithCopyEnabled = {
  args: {
    enableCopy: true,
  },
};

export const WithCopyEnabledInsideInput = {
  args: {
    enableCopy: true,
    copyButtonPosition: "inside",
  },
};
