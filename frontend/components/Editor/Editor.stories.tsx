import { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";

import Editor from ".";

const meta: Meta<typeof Editor> = {
  component: Editor,
  title: "Components/FormFields/Editor",
  argTypes: {
    mode: {
      control: "select",
      options: ["sh", "powershell"],
      description: "Syntax highlighting mode",
    },
    readOnly: { control: "boolean" },
    enableCopy: { control: "boolean" },
    wrapEnabled: { control: "boolean" },
    isFormField: { control: "boolean" },
    focus: { control: "boolean" },
    label: { control: "text" },
    labelTooltip: { control: "text" },
    error: { control: "text" },
    helpText: { control: "text" },
    value: { control: "text" },
    defaultValue: { control: "text" },
    maxLines: { control: "number" },
    name: { control: "text" },
  },
};

export default meta;

type Story = StoryObj<typeof Editor>;

export const Default: Story = {
  args: {
    name: "default-editor",
    label: "Shell Script Editor",
    value: "#!/bin/bash\necho 'Hello, World!'",
    mode: "sh",
    onChange: action("onChange"),
    onBlur: action("onBlur"),
  },
};

export const WithError: Story = {
  args: {
    ...Default.args,
    name: "error-editor",
    label: "Editor with Error",
    error: "There is a syntax error",
    value: "echo 'Missing closing quote",
  },
};

export const WithHelpText: Story = {
  args: {
    ...Default.args,
    name: "help-text-editor",
    label: "Editor with Help Text",
    helpText: "Write your shell script here. Supports Bash and PowerShell.",
  },
};

export const WithTooltip: Story = {
  args: {
    ...Default.args,
    name: "tooltip-editor",
    label: "Editor with Tooltip",
    labelTooltip: "This editor supports syntax highlighting for shell scripts.",
  },
};

export const ReadOnly: Story = {
  args: {
    ...Default.args,
    name: "readonly-editor",
    label: "Read-only Editor",
    readOnly: true,
    value: "# This editor is read-only\nls -la",
  },
};

export const WithCopyButton: Story = {
  args: {
    ...Default.args,
    name: "copy-editor",
    label: "Editor with Copy Button",
    value: "echo 'Copy this script!'",
    enableCopy: true,
  },
};

export const PowerShellMode: Story = {
  args: {
    ...Default.args,
    name: "powershell-editor",
    label: "PowerShell Editor",
    value: "Write-Host 'Hello from PowerShell!'",
    mode: "powershell",
  },
};

export const WrappedLines: Story = {
  args: {
    ...Default.args,
    name: "wrapped-lines-editor",
    label: "Editor with Wrapped Lines",
    value:
      "# This is a very long line that should wrap to the next line in the editor to demonstrate the wrapEnabled prop in action.",
    wrapEnabled: true,
  },
};

export const CustomMaxLines: Story = {
  args: {
    ...Default.args,
    name: "custom-maxlines-editor",
    label: "Editor with Custom Max Lines",
    value: "echo 'Line 1'\necho 'Line 2'\necho 'Line 3'\n",
    maxLines: 3,
  },
};

export const LongScriptWithCopy: Story = {
  args: {
    name: "long-script-editor",
    label: "Long Script with Copy Button",
    enableCopy: true,
    mode: "sh",
    value: `#!/bin/bash
# This is a long example script to test the editor's UI with overflow and copy button interaction.

echo "Starting system update..."
sudo apt-get update -y && sudo apt-get upgrade -y

echo "Installing dependencies..."
sudo apt-get install -y git curl wget unzip build-essential python3 python3-pip

echo "Cloning repository..."
git clone https://github.com/example/repo.git /opt/example-repo

echo "Setting up environment variables..."
export APP_ENV=production
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=admin
export DB_PASSWORD=supersecurepassword1234567890

echo "Configuring application..."
cd /opt/example-repo
cp config.example.json config.json
sed -i 's/localhost/127.0.0.1/g' config.json

echo "Running database migrations..."
python3 manage.py migrate

echo "Starting application..."
nohup python3 manage.py runserver 0.0.0.0:8000 &

echo "Setup complete! Application is running."
`,
    wrapEnabled: false, // Try toggling this to true to see the difference!
    maxLines: 20,
    helpText:
      "This is a realistic, long shell script. Try copying it or scrolling horizontally.",
    onChange: action("onChange"),
    onBlur: action("onBlur"),
  },
};
