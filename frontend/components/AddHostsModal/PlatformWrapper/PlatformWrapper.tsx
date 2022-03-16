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
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import TabsWrapper from "components/TabsWrapper";

import { isValidPemCertificate } from "../../../pages/hosts/ManageHostsPage/helpers";

import CopyIcon from "../../../../assets/images/icon-copy-clipboard-fleet-blue-20x20@2x.png";
import DownloadIcon from "../../../../assets/images/icon-download-12x12@2x.png";

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
    name: "Linux (deb)",
    type: "deb",
  },
  {
    name: "Advanced",
    type: "advanced",
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
  const { config, isPreviewMode } = useContext(AppContext);
  const [copyMessage, setCopyMessage] = useState<string>("");
  const [includeFleetDesktop, setIncludeFleetDesktop] = useState<boolean>(
    false
  );

  const dispatch = useDispatch();

  const {
    data: certificate,
    error: fetchCertificateError,
    isFetching: isFetchingCertificate,
  } = useQuery<string, Error>(
    ["certificate"],
    () => configAPI.loadCertificate(),
    {
      enabled: !isPreviewMode,
      refetchOnWindowFocus: false,
    }
  );

  let tlsHostname = config?.server_url || "";

  try {
    const serverUrl = new URL(config?.server_url || "");
    tlsHostname = serverUrl.hostname;
    if (serverUrl.port) {
      tlsHostname += `:${serverUrl.port}`;
    }
  } catch (e) {
    if (!(e instanceof TypeError)) {
      throw e;
    }
  }

  const flagfileContent = `# Server
--tls_hostname=${tlsHostname}
--tls_server_certs=fleet.pem
# Enrollment
--host_identifier=instance
--enroll_secret_path=secret.txt
--enroll_tls_endpoint=/api/v1/osquery/enroll
# Configuration
--config_plugin=tls
--config_tls_endpoint=/api/v1/osquery/config
--config_refresh=10
# Live query
--disable_distributed=false
--distributed_plugin=tls
--distributed_interval=10
--distributed_tls_max_attempts=3
--distributed_tls_read_endpoint=/api/v1/osquery/distributed/read
--distributed_tls_write_endpoint=/api/v1/osquery/distributed/write
# Logging
--logger_plugin=tls
--logger_tls_endpoint=/api/v1/osquery/log
--logger_tls_period=10
# File carving
--disable_carver=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=2000000`;

  let enrollSecret: string;
  if (selectedTeam.secrets) {
    enrollSecret = selectedTeam.secrets[0].secret;
  }

  const onDownloadEnrollSecret = (evt: React.MouseEvent) => {
    evt.preventDefault();

    const filename = "secret.txt";
    const file = new global.window.File([enrollSecret], filename);

    FileSaver.saveAs(file);

    return false;
  };

  const onDownloadFlagfile = (evt: React.MouseEvent) => {
    evt.preventDefault();

    const filename = "flagfile.txt";
    const file = new global.window.File([flagfileContent], filename);

    FileSaver.saveAs(file);

    return false;
  };

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
    return platform === "advanced"
      ? "osqueryd --flagfile=flagfile.txt --verbose"
      : `fleetctl package --type=${platform} ${
          includeFleetDesktop ? "--fleet-desktop " : ""
        }--fleet-url=${config?.server_url} --enroll-secret=${enrollSecret}`;
  };

  const renderLabel = (platform: string, installerString: string) => {
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
        {platform !== "advanced" && (
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
          </span>
        )}{" "}
        <span className={`${baseClass}__name`}>
          <span className="buttons">
            {copyMessage && (
              <span
                className={`${baseClass}__copy-message`}
              >{`${copyMessage} `}</span>
            )}
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
    if (platform === "advanced") {
      return (
        <div className={baseClass}>
          <div className={`${baseClass}__advanced`}>
            <div className={`${baseClass}__advanced--enroll-secrets`}>
              <p className={`${baseClass}__advanced--heading`}>
                Download your enroll secret:
              </p>
              <p>
                Osquery uses an enroll secret to authenticate with the Fleet
                server.
                <br />
                <a
                  href="#downloadEnrollSecret"
                  onClick={onDownloadEnrollSecret}
                >
                  Download
                  <img src={DownloadIcon} alt="download icon" />
                </a>
              </p>
            </div>
            <div className={`${baseClass}__advanced--fleet-certificate`}>
              <p className={`${baseClass}__advanced--heading`}>
                Download your Fleet certificate:
              </p>
              {isFetchingCertificate && (
                <p className={`${baseClass}__certificate-loading`}>
                  Loading your certificate
                </p>
              )}
              {!isFetchingCertificate &&
                (certificate ? (
                  <p>
                    Prove the TLS certificate used by the Fleet server to enable
                    secure connections from osquery:
                    <br />
                    <a
                      href="#downloadCertificate"
                      className={`${baseClass}__fleet-certificate-download`}
                      onClick={onDownloadCertificate}
                    >
                      Download
                      <img src={DownloadIcon} alt="download" />
                    </a>
                  </p>
                ) : (
                  <p className={`${baseClass}__certificate-error`}>
                    <em>Fleet failed to load your certificate.</em>
                    <span>
                      If you&apos;re able to access Fleet at a private or secure
                      (HTTPS) IP address, please log into Fleet at this address
                      to load your certificate.
                    </span>
                  </p>
                ))}
            </div>
            <div className={`${baseClass}__advanced--enroll-secrets`}>
              <p className={`${baseClass}__advanced--heading`}>
                Download your flagfile:
              </p>
              <p>
                If using the enroll secret and server certificate downloaded
                above, us the generated flagfile. In some configurations,
                modifications may need to be made.
                <br />
                {fetchCertificateError ? (
                  <span className={`${baseClass}__error`}>
                    {fetchCertificateError}
                  </span>
                ) : (
                  <a href="#downloadFlagfile" onClick={onDownloadFlagfile}>
                    Download
                    <img src={DownloadIcon} alt="download icon" />
                  </a>
                )}
              </p>
            </div>
            <div className={`${baseClass}__advanced--enroll-secrets`}>
              <p className={`${baseClass}__advanced--heading`}>
                With{" "}
                <a
                  href="https://www.osquery.io/downloads"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  osquery
                </a>{" "}
                installed:
              </p>
              <p>
                Run osquery from the directory containing the above files (may
                require sudo or Run as Administrator privileges):
                <InputField
                  disabled
                  inputWrapperClass={`${baseClass}__run-osquery-input`}
                  name="run-osquery"
                  label={renderLabel(platform, renderInstallerString(platform))}
                  type={"text"}
                  value={renderInstallerString(platform)}
                />
              </p>
            </div>
          </div>
        </div>
      );
    }
    return (
      <>
        <Checkbox
          name="include-fleet-desktop"
          onChange={() => setIncludeFleetDesktop(!includeFleetDesktop)}
          value={includeFleetDesktop}
        >
          <>
            Include&nbsp;
            <TooltipWrapper
              tipContent={
                "<p>Lightweight application that allows end users to see information about their device.</p>"
              }
            >
              Fleet Desktop
            </TooltipWrapper>
          </>
        </Checkbox>
        <InputField
          disabled
          inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${platform}`}
          name="installer"
          label={renderLabel(platform, renderInstallerString(platform))}
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
