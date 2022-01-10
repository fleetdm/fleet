import React, { useContext, useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { useQuery } from "react-query";
import FileSaver from "file-saver";

import { useDispatch } from "react-redux";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import configAPI from "services/entities/config";
import { AppContext } from "context/app";
// @ts-ignore
import { stringToClipboard } from "utilities/copy_text";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import TabsWrapper from "components/TabsWrapper";

import { isValidPemCertificate } from "../../../helpers";

import CopyIcon from "../../../../../../../assets/images/icon-copy-clipboard-fleet-blue-20x20@2x.png";
import DownloadIcon from "../../../../../../../assets/images/icon-download-12x12@2x.png";

interface IPlatformSubNav {
  name: string;
  type: string;
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
  selectedTeam: ITeam | { name: string; secrets: IEnrollSecret[] | null };
  onCancel: () => void;
}

const baseClass = "platform-wrapper";

const PlatformWrapper = ({
  selectedTeam,
  onCancel,
}: IPlatformWrapperProp): JSX.Element => {
  const { config } = useContext(AppContext);
  const [copyMessage, setCopyMessage] = useState<string>("");

  const dispatch = useDispatch();

  const { data: certificate, isFetching: isFetchingCertificate } = useQuery<
    string,
    Error
  >(["certificate"], () => configAPI.loadCertificate(), {
    refetchOnWindowFocus: false,
  });

  const onDownloadCertificate = (evt: React.MouseEvent) => {
    evt.preventDefault();

    if (certificate && isValidPemCertificate(certificate)) {
      const filename = "fleet.pem";
      const file = new global.window.File([certificate], filename, {
        type: "application/x-pem-file",
      });

      FileSaver.saveAs(file);
    } else {
      dispatch(
        renderFlash(
          "error",
          "Your certificate could not be downloaded. Please check your Fleet configuration."
        )
      );
    }
    return false;
  };

  const renderInstallerString = (platform: string) => {
    let enrollSecret;
    if (selectedTeam.secrets) {
      enrollSecret = selectedTeam.secrets[0].secret;
    }

    let installerString = `fleetctl package --type=${platform} --fleet-url=${config?.server_url} --enroll-secret=${enrollSecret}`;
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
            <p className={`${baseClass}__cta`}>
              Download your Fleet certificate:
            </p>
            {isFetchingCertificate && (
              <p className={`${baseClass}__certificate-loading`}>
                Loading your certificate
              </p>
            )}
            {!isFetchingCertificate &&
              (certificate ? (
                <a
                  href="#downloadCertificate"
                  className={`${baseClass}__fleet-certificate-download`}
                  onClick={onDownloadCertificate}
                >
                  Download
                  <img src={DownloadIcon} alt="download" />
                </a>
              ) : (
                <p className={`${baseClass}__certificate-error`}>
                  <em>Fleet failed to load your certificate.</em>
                  <span>
                    If you&apos;re able to access Fleet at a private or secure
                    (HTTPS) IP address, please log into Fleet at this address to
                    load your certificate.
                  </span>
                </p>
              ))}
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
      <TabsWrapper>
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
      </TabsWrapper>
      <div className={`${baseClass}__button-wrap`}>
        <Button onClick={onCancel} className="button button--brand">
          Done
        </Button>
      </div>
    </div>
  );
};

export default PlatformWrapper;
