import React, { useCallback, useContext, useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import FileSaver from "file-saver";

import fleetdPackageAPI, {
  FleetdPackageLinux,
} from "services/entities/fleetd_package";

import { NotificationContext } from "context/notification";
import { IConfig } from "interfaces/config";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import RevealButton from "components/buttons/RevealButton";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";
import TabNav from "components/TabNav";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink/CustomLink";
import Radio from "components/forms/fields/Radio";
import TabText from "components/TabText";

import { isValidPemCertificate } from "../../../pages/hosts/ManageHostsPage/helpers";
import IosIpadosPanel from "./IosIpadosPanel";
import AndroidPanel from "./AndroidPanel";

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
    name: "Linux",
    type: "deb",
  },
  {
    name: "ChromeOS",
    type: "chromeos",
  },
  {
    name: "iOS & iPadOS",
    type: "ios-ipados",
  },
  {
    name: "Android",
    type: "android",
  },
  {
    name: "Advanced",
    type: "advanced",
  },
];

interface IPlatformWrapperProps {
  enrollSecret: string;
  onCancel: () => void;
  certificate: any;
  isFetchingCertificate: boolean;
  fetchCertificateError: any;
  config: IConfig | null;
}

const baseClass = "platform-wrapper";

const PlatformWrapper = ({
  enrollSecret,
  onCancel,
  certificate,
  isFetchingCertificate,
  fetchCertificateError,
  config,
}: IPlatformWrapperProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [hostType, setHostType] = useState<"workstation" | "server">(
    "workstation"
  );
  const [archType, setArchType] = useState<"amd64" | "arm64">("amd64");
  const [pkgType, setPkgType] = useState<FleetdPackageLinux>("deb");
  const [showPlainOsquery, setShowPlainOsquery] = useState(false);
  const [selectedTabIndex, setSelectedTabIndex] = useState(0); // External link requires control in state
  const [isLoadingFleetdPkg, setIsLoadingFleetdPkg] = useState(false);

  let tlsHostname = config?.server_settings.server_url || "";

  try {
    const serverUrl = new URL(config?.server_settings.server_url || "");
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
--enroll_tls_endpoint=/api/osquery/enroll
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
--carver_block_size=8000000`;

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
      renderFlash(
        "error",
        "Your certificate could not be downloaded. Please check your Fleet configuration."
      );
    }
    return false;
  };

  const renderFleetCertificateBlock = (type: "plain" | "tooltip") => {
    return (
      <div className={`${baseClass}__advanced--fleet-certificate`}>
        {type === "plain" ? (
          <div className={`${baseClass}__advanced--heading`}>
            Download your Fleet certificate
          </div>
        ) : (
          <div
            className={`${baseClass}__advanced--heading download-certificate--tooltip`}
          >
            Download your{" "}
            <TooltipWrapper tipContent="A Fleet certificate is required if Fleet is running with a self signed or otherwise untrusted certificate.">
              Fleet certificate:
            </TooltipWrapper>
          </div>
        )}
        {isFetchingCertificate && (
          <p className={`${baseClass}__certificate-loading`}>
            Loading your certificate
          </p>
        )}
        {!isFetchingCertificate &&
          (certificate ? (
            <p>
              {type === "plain" && (
                <>
                  Prove the TLS certificate used by the Fleet server to enable
                  secure connections from osquery:
                  <br />
                </>
              )}
              <Button
                variant="text-icon"
                className={`${baseClass}__fleet-certificate-download`}
                onClick={onDownloadCertificate}
              >
                Download
                <Icon name="download" color="core-fleet-blue" size="small" />
              </Button>
            </p>
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
      </div>
    );
  };

  const renderInstallerString = (packageType: string) => {
    return packageType === "advanced"
      ? `fleetctl package --type=YOUR_TYPE --fleet-url=${config?.server_settings.server_url} --enroll-secret=${enrollSecret} --fleet-certificate=PATH_TO_YOUR_CERTIFICATE/fleet.pem`
      : `fleetctl package --type=${packageType} ${
          config && !config.server_settings.scripts_disabled
            ? "--enable-scripts "
            : ""
        }${hostType === "workstation" ? "--fleet-desktop " : ""}--fleet-url=${
          config?.server_settings.server_url
        } --enroll-secret=${enrollSecret}`;
  };

  const renderLabel = (packageType: string) => {
    return (
      <>
        {packageType !== "plain-osquery" && (
          <span className={`${baseClass}__cta`}>
            Run this command with the{" "}
            <a
              className={`${baseClass}__command-line-tool`}
              href="https://fleetdm.com/learn-more-about/installing-fleetctl"
              target="_blank"
              rel="noopener noreferrer"
            >
              Fleet command-line tool
            </a>
            :
          </span>
        )}
      </>
    );
  };


  const renderPanel = (packageType: string) => {
    const CHROME_OS_INFO = {
      extensionId: "fleeedmmihkfkeemmipgmhhjemlljidg",
      installationUrl: "https://chrome.fleetdm.com/updates.xml",
      policyForExtension: `{
  "fleet_url": {
    "Value": "${config?.server_settings.server_url}"
  },
  "enroll_secret": {
    "Value": "${enrollSecret}"
  }
}`,
    };

    let packageTypeHelpText;
    if (packageType === "deb") {
      packageTypeHelpText = (
        <>
          For CentOS, Red Hat, and Fedora Linux, use <code>--type=rpm</code>.
          For ARM, use <code>--arch=arm64</code>
        </>
      );
    } else if (packageType === "msi") {
      packageTypeHelpText = (
        <>
          For ARM, use <code>--arch=arm64</code>
        </>
      );
    } else if (packageType === "pkg") {
      packageTypeHelpText = "Install this package to add hosts to Fleet.";
    } else {
      packageTypeHelpText = "";
    }

    if (packageType === "chromeos") {
      return (
        <>
          <div className={`${baseClass}__chromeos--info`}>
            <p className={`${baseClass}__chromeos--heading`}>
              In Google Admin:
            </p>
            <p>
              Add the extension for the relevant users & browsers using the
              information below.
            </p>
            <InfoBanner className={`${baseClass}__chromeos--instructions`}>
              For a step-by-step guide, see the documentation page for{" "}
              <CustomLink
                url="https://fleetdm.com/docs/using-fleet/adding-hosts#enroll-chromebooks"
                text="adding hosts"
                newTab
                multiline
              />
            </InfoBanner>
          </div>
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__chromeos-extension-id`}
            name="Extension ID"
            enableCopy
            label="Extension ID"
            value={CHROME_OS_INFO.extensionId}
          />
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__chromeos-url`}
            name="Installation URL"
            enableCopy
            label="Installation URL"
            value={CHROME_OS_INFO.installationUrl}
          />
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__chromeos-policy-for-extension`}
            name="Policy for extension"
            enableCopy
            label="Policy for extension"
            type="textarea"
            value={CHROME_OS_INFO.policyForExtension}
          />
        </>
      );
    }

    if (packageType === "ios-ipados") {
      return <IosIpadosPanel enrollSecret={enrollSecret} />;
    }

    if (packageType === "android") {
      return <AndroidPanel enrollSecret={enrollSecret} />;
    }

    if (packageType === "advanced") {
      return (
        <>
          {renderFleetCertificateBlock("tooltip")}
          <div className={`${baseClass}__advanced--installer`}>
            <InputField
              readOnly
              inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${packageType}`}
              name="installer"
              enableCopy
              label={renderLabel(packageType)}
              type="textarea"
              value={renderInstallerString(packageType)}
              helpText="Distribute your package to add hosts to Fleet."
            />
          </div>
          <div>
            <InfoBanner className={`${baseClass}__chrome--instructions`}>
              This works for macOS, Windows, and Linux hosts. To add
              Chromebooks,{" "}
              <Button
                variant="text-link"
                onClick={() => setSelectedTabIndex(4)}
              >
                click here
              </Button>
              .
            </InfoBanner>
          </div>
          <RevealButton
            className={baseClass}
            isShowing={showPlainOsquery}
            hideText="Plain osquery"
            showText="Plain osquery"
            caretPosition="after"
            onClick={() => setShowPlainOsquery((prev) => !prev)}
          />
          {showPlainOsquery && (
            <>
              <div className={`${baseClass}__advanced--enroll-secrets`}>
                <p className={`${baseClass}__advanced--heading`}>
                  Download your enroll secret:
                </p>
                <p>
                  Osquery uses an enroll secret to authenticate with the Fleet
                  server.
                  <br />
                  <Button variant="text-icon" onClick={onDownloadEnrollSecret}>
                    Download
                    <Icon
                      name="download"
                      color="core-fleet-blue"
                      size="small"
                    />
                  </Button>
                </p>
              </div>
              {renderFleetCertificateBlock("plain")}
              <div className={`${baseClass}__advanced--flagfile`}>
                <p className={`${baseClass}__advanced--heading`}>
                  Download your flagfile:
                </p>
                <p>
                  If using the enroll secret and server certificate downloaded
                  above, use the generated flagfile. In some configurations,
                  modifications may need to be made.
                  <br />
                  {fetchCertificateError ? (
                    <span className={`${baseClass}__error`}>
                      {fetchCertificateError}
                    </span>
                  ) : (
                    <Button variant="text-icon" onClick={onDownloadFlagfile}>
                      Download
                      <Icon
                        name="download"
                        color="core-fleet-blue"
                        size="small"
                      />
                    </Button>
                  )}
                </p>
              </div>
              <div className={`${baseClass}__advanced--osqueryd`}>
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
                <p className={`${baseClass}__advanced--text`}>
                  Run osquery from the directory containing the above files (may
                  require sudo or Run as Administrator privileges):
                </p>
                <InputField
                  readOnly
                  inputWrapperClass={`${baseClass}__run-osquery-input`}
                  name="run-osquery"
                  enableCopy
                  label={renderLabel("plain-osquery")}
                  type="text"
                  value="osqueryd --flagfile=flagfile.txt --verbose"
                />
              </div>
            </>
          )}
        </>
      );
    }

    const onDownloadFleetdLinuxPackage = async () => {
      setIsLoadingFleetdPkg(true);
      try {
        const content = await fleetdPackageAPI.load(
          pkgType,
          archType,
          hostType === "workstation"
        );
        const filename = `fleet-osquery-${archType}.${pkgType}`;
        const file = new File([content], filename);
        FileSaver.saveAs(file);
      } catch {
        renderFlash("error", "Couldn't download. Please try again.");
      } finally {
        setIsLoadingFleetdPkg(false);
      }
    };

    if (packageType === "pkg") {
      return (
        <div className="form">
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${packageType}`}
            name="installer"
            enableCopy
            label={renderLabel(packageType)}
            type="textarea"
            value={renderInstallerString(packageType)}
            helpText={packageTypeHelpText}
          />
        </div>
      );
    }

    if (packageType === "deb" || packageType === "rpm") {
      return (
        // "form" className applies the global form styling
        <div>
          <div className="form">
            <div className="form-field">
              <div className="form-field__label">Host type</div>
              <Radio
                className={`${baseClass}__radio-input`}
                label={
                  <TooltipWrapper tipContent="Contains the Fleet Desktop tray application.">
                    <span>Workstation</span>
                  </TooltipWrapper>
                }
                id="workstation-host"
                checked={hostType === "workstation"}
                value="workstation"
                name="host-typ"
                onChange={() => setHostType("workstation")}
              />
              <Radio
                className={`${baseClass}__radio-input`}
                label="Server"
                id="server-host"
                checked={hostType === "server"}
                value="server"
                name="host-type"
                onChange={() => setHostType("server")}
              />
            </div>
            <div className="form-field">
              <div className="form-field__label">Package type</div>
              <Radio
                className={`${baseClass}__radio-input`}
                label="deb"
                id="deb-type"
                checked={pkgType === "deb"}
                value="deb"
                name="pkg-type"
                onChange={() => setPkgType("deb")}
              />
              <Radio
                className={`${baseClass}__radio-input`}
                label="rpm"
                id="rpm-type"
                checked={pkgType === "rpm"}
                value="rpm"
                name="pkg-type"
                onChange={() => setPkgType("rpm")}
              />
            </div>
            <div className="form-field">
              <div className="form-field__label">Package architecture</div>
              <Radio
                className={`${baseClass}__radio-input`}
                label="amd64"
                id="amd64-type"
                checked={archType === "amd64"}
                value="amd64"
                name="arch-type"
                onChange={() => setArchType("amd64")}
              />
              <Radio
                className={`${baseClass}__radio-input`}
                label="arm64"
                id="arm64-type"
                checked={archType === "arm64"}
                value="arm64"
                name="arch-type"
                onChange={() => setArchType("arm64")}
              />
            </div>
          </div>
          <div className="modal-cta-wrap">
            <Button
              onClick={onDownloadFleetdLinuxPackage}
              isLoading={isLoadingFleetdPkg}
              disabled={isLoadingFleetdPkg}
            >
              Download
            </Button>
          </div>
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${packageType}`}
            name="installer-linux"
            enableCopy
            label={renderLabel(packageType)}
            type="textarea"
            value={renderInstallerString(packageType)}
            helpText={packageTypeHelpText}
          />
        </div>
      );
    }

    if (packageType === "msi") {
      return (
        // "form" className applies the global form styling
        <div className="form">
          <div className="form-field">
            <div className="form-field__label">Type</div>
            <Radio
              className={`${baseClass}__radio-input`}
              label="Workstation"
              id="workstation-host"
              checked={hostType === "workstation"}
              value="workstation"
              name="host-typ"
              onChange={() => setHostType("workstation")}
            />
            <Radio
              className={`${baseClass}__radio-input`}
              label="Server"
              id="server-host"
              checked={hostType === "server"}
              value="server"
              name="host-type"
              onChange={() => setHostType("server")}
            />
          </div>
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${packageType}`}
            name="installer"
            enableCopy
            label={renderLabel(packageType)}
            type="textarea"
            value={renderInstallerString(packageType)}
            helpText={packageTypeHelpText}
          />
        </div>
      );
    }
  };

  return (
    <div className={baseClass}>
      <TabNav>
        <Tabs
          onSelect={(index) => setSelectedTabIndex(index)}
          selectedIndex={selectedTabIndex}
        >
          <TabList>
            {platformSubNav.map((navItem) => {
              // Bolding text when the tab is active causes a layout shift
              // so we add a hidden pseudo element with the same text string
              return (
                <Tab key={navItem.name} data-text={navItem.name}>
                  <TabText>{navItem.name}</TabText>
                </Tab>
              );
            })}
          </TabList>
          {platformSubNav.map((navItem) => {
            // Bolding text when the tab is active causes a layout shift
            // so we add a hidden pseudo element with the same text string
            return (
              <TabPanel className={`${baseClass}__info`} key={navItem.type}>
                <div className={`${baseClass} form`}>
                  {renderPanel(navItem.type)}
                </div>
              </TabPanel>
            );
          })}
        </Tabs>
      </TabNav>
    </div>
  );
};

export default PlatformWrapper;
