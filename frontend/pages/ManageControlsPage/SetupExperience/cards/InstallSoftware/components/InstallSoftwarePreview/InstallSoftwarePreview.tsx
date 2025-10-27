import React from "react";

import Card from "components/Card";

import { SetupExperiencePlatform } from "interfaces/platform";

import LinuxAndWindowsInstallSoftwareEndUserPreview from "../../../../../../../../assets/videos/linux-windows-install-software-preview.mp4";
import MacInstallSoftwareEndUserPreview from "../../../../../../../../assets/videos/mac-install-software-preview.mp4";

const baseClass = "install-software-preview";

interface IPreviewDisplayConfig {
  description: React.ReactNode;
  videoSrc: string;
}

const PREVIEW_DISPLAY_OPTIONS: Record<
  SetupExperiencePlatform,
  IPreviewDisplayConfig | undefined
> = {
  macos: {
    description: (
      <>
        <p>
          During the <b>Remote Management</b> screen, the end user will see
          selected software being installed. They won&apos;t be able to continue
          until software is installed.
        </p>
        <p>
          If there are any errors, they will be able to continue and will be
          instructed to contact their IT admin.
        </p>
      </>
    ),
    videoSrc: MacInstallSoftwareEndUserPreview,
  },
  ios: undefined,
  ipados: undefined,
  linux: {
    description: (
      <>
        <p>
          When Fleet&apos;s agent (fleetd) is installed, fleetd will open the{" "}
          <b>Fleet Desktop &gt; My device</b> page in the default browser.
        </p>
        <p>The end user will see selected software being installed.</p>
      </>
    ),
    videoSrc: LinuxAndWindowsInstallSoftwareEndUserPreview,
  },
  windows: {
    description: (
      <>
        <p>
          When Fleet&apos;s agent (fleetd) is installed, fleetd will open the{" "}
          <b>Fleet Desktop &gt; My device</b> page in the default browser.
        </p>
        <p>The end user will see selected software being installed.</p>
      </>
    ),
    videoSrc: LinuxAndWindowsInstallSoftwareEndUserPreview,
  },
};

interface InstallSoftwarePreviewProps {
  platform: SetupExperiencePlatform;
}

const InstallSoftwarePreview = ({ platform }: InstallSoftwarePreviewProps) => {
  const { description, videoSrc } = PREVIEW_DISPLAY_OPTIONS[platform] || {};
  return description && videoSrc ? (
    <Card color="grey" paddingSize="xxlarge" className={baseClass}>
      <h3>End user experience</h3>
      {description}
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <video
        className={`${baseClass}__preview-video`}
        src={videoSrc}
        controls
        autoPlay
        loop
        muted
      />
    </Card>
  ) : null;
};

export default InstallSoftwarePreview;
