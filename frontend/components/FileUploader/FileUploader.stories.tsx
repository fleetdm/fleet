import { Meta, StoryObj } from "@storybook/react";

import FileUploader from "./FileUploader";

const meta: Meta<typeof FileUploader> = {
  title: "Components/FileUploader",
  component: FileUploader,
  args: {
    graphicNames: "file-configuration-profile",
    message: "The main message",
    additionalInfo: "The additional message",
    accept: ".pdf",
    isLoading: false,
    onFileUpload: () => {
      alert("File uploaded!");
    },
  },
};

export default meta;

type Story = StoryObj<typeof FileUploader>;

export const Basic: Story = {};
