import React, { useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { useDispatch, useSelector } from "react-redux";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import { stringToClipboard } from "utilities/copy_text";
import FileSaver from "file-saver";
import { IConfig } from "interfaces/config";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";
import CopyIcon from "../../../../../../../../assets/images/icon-copy-clipboard-fleet-blue-20x20@2x.png";
import DownloadIcon from "../../../../../../../../assets/images/icon-download-12x12@2x.png";

interface IPlatformSubNav {
  name: string;
  type: string;
}

interface IRootState {
  app: {
    config: IConfig;
  };
}

const platformSubNav: IPlatformSubNav[] = [
  {
    name: "macOS",
    type: "pkg",
  },
  {
    name: "Windows",
    type: "msi",
  },
  {
    name: "Linux (RPM)",
    type: "rpm",
  },
  {
    name: "Linux (DEB)",
    type: "deb",
  },
];

interface IPlatformWrapperProp {
  selectedTeam: ITeam | { name: string; secrets: IEnrollSecret[] };
  certificate: string;
  onCancel: () => void;
}

const baseClass = "platform-wrapper";

const PlatformWrapper = ({
  selectedTeam,
  certificate,
  onCancel,
}: IPlatformWrapperProp): JSX.Element => {
  console.log("selectedTeam", selectedTeam);

  const [copyMessage, setCopyMessage] = useState<string>("");

  const onDownloadCertificate = (evt: React.MouseEvent) => {
    evt.preventDefault();

    const filename = "fleet-certificate.txt";
    const file = new global.window.File([certificate], filename);

    FileSaver.saveAs(file);

    return false;
  };

  const renderInstallerString = (platform: string) => {
    let enrollSecret;
    if (selectedTeam.secrets) {
      enrollSecret = selectedTeam.secrets[0].secret;
    }

    let installerString = `fleetctl package --type=${platform} --fleet-url=https://localhost:8412 --enroll-secret=${enrollSecret}`;
    if (platform === "rpm" || platform === "deb") {
      installerString +=
        " --fleet-certificate=/home/username/Downloads/fleet.pem";
    }
    return installerString;
  };

  const renderLabel = (installerString: string) => {
    const onCopyInstaller = (evt: React.MouseEvent) => {
      evt.preventDefault();

      stringToClipboard(installerString)
        .then(() => setCopyMessage("Copied!"))
        .catch(() => setCopyMessage("Copy failed"));

      // Clear message after 1 second
      setTimeout(() => setCopyMessage(""), 1000);

      return false;
    };

    return (
      <>
        <span className={`${baseClass}__cta`}>
          With the{" "}
          <a
            href="https://fleetdm.com/get-started"
            target="_blank"
            rel="noopener noreferrer"
            className={`${baseClass}__command-line-tool`}
          >
            Fleet command-line tool
          </a>{" "}
          installed:
        </span>{" "}
        <span className={`${baseClass}__name`}>
          <span className="buttons">
            {copyMessage && <span>{`${copyMessage} `}</span>}
            <Button
              variant="unstyled"
              className={`${baseClass}__installer-copy-icon`}
              onClick={onCopyInstaller}
            >
              <img src={CopyIcon} alt="copy" />
            </Button>
          </span>
        </span>
      </>
    );
  };

  const renderTab = (platform: string) => {
    return (
      <>
        {(platform === "rpm" || platform === "deb") && (
          <>
            <span className={`${baseClass}__cta`}>
              Download your Fleet certificate:
            </span>
            <p>
              <a
                href="#onDownloadCertificate"
                className={`${baseClass}__fleet-certificate-download`}
                onClick={onDownloadCertificate}
              >
                Download
                <img src={DownloadIcon} alt="download" />
              </a>
            </p>
          </>
        )}
        <InputField
          disabled
          inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${platform}`}
          name="installer"
          label={renderLabel(renderInstallerString(platform))}
          type={"textarea"}
          value={renderInstallerString(platform)}
        />
        <span>
          Generates an installer that your devices will use to connect to Fleet.
        </span>
      </>
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__nav-header`}>
        <Tabs>
          <TabList>
            {platformSubNav.map((navItem) => {
              // Bolding text when the tab is active causes a layout shift
              // so we add a hidden pseudo element with the same text string
              return (
                <Tab key={navItem.name} data-text={navItem.name}>
                  {navItem.name}
                </Tab>
              );
            })}
          </TabList>
          {platformSubNav.map((navItem) => {
            // Bolding text when the tab is active causes a layout shift
            // so we add a hidden pseudo element with the same text string
            return (
              <TabPanel className={`${baseClass}__info`} key={navItem.type}>
                {renderTab(navItem.type)}
              </TabPanel>
            );
          })}
        </Tabs>
      </div>
      <div className={`${baseClass}__button-wrap`}>
        <Button onClick={onCancel} className="button button--brand">
          Done
        </Button>
      </div>
    </div>
  );
};

export default PlatformWrapper;
