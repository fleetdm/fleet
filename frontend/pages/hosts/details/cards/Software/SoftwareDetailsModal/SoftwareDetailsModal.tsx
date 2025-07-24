import React, { useState, useRef, useEffect } from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import {
  IHostSoftware,
  ISoftwareInstallVersion,
  SoftwareSource,
  formatSoftwareType,
  hasHostSoftwareAppLastInstall,
  hasHostSoftwarePackageLastInstall,
} from "interfaces/software";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import Card from "components/Card";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import { dateAgo } from "utilities/date_format";

import { AppInstallDetails } from "components/ActivityDetails/InstallDetails/AppInstallDetails";
import { SoftwareInstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareInstallDetailsModal";

const baseClass = "software-details-modal";

const generateVulnerabilitiesValue = (vulnerabilities: string[]) => {
  const first3 = vulnerabilities.slice(0, 3);
  const rest = vulnerabilities.slice(3);

  const first3Text = first3.join(", ");
  const restText = `, +${rest.length} more`;

  return (
    <>
      <span>{`${first3Text}${rest.length > 0 ? restText : ""}`}</span>
    </>
  );
};

interface ISoftwareDetailsInfoProps {
  installedVersion: ISoftwareInstallVersion;
  source: SoftwareSource;
  bundleIdentifier?: string;
}

const SoftwareDetailsInfo = ({
  installedVersion,
  source,
  bundleIdentifier,
}: ISoftwareDetailsInfoProps) => {
  const {
    vulnerabilities,
    installed_paths: installedPaths,
    signature_information: signatureInformation,
  } = installedVersion;

  return (
    <Card
      className={`${baseClass}__version-details`}
      color="grey"
      borderRadiusSize="medium"
    >
      <div className={`${baseClass}__row`}>
        <DataSet title="Version" value={installedVersion.version} />
        <DataSet title="Type" value={formatSoftwareType({ source })} />
        {bundleIdentifier && (
          <DataSet title="Bundle identifier" value={bundleIdentifier} />
        )}
        {installedVersion.last_opened_at && (
          <DataSet
            title="Last used"
            value={dateAgo(installedVersion.last_opened_at)}
          />
        )}
      </div>
      {vulnerabilities && vulnerabilities.length !== 0 && (
        <div className={`${baseClass}__row`}>
          <DataSet
            title="Vulnerabilities"
            value={generateVulnerabilitiesValue(vulnerabilities)}
          />
        </div>
      )}
      {!!installedPaths?.length &&
        installedPaths.map((path) => {
          // Find the signature info for this path
          const sigInfo = signatureInformation?.find(
            (info) => info.installed_path === path
          );

          return (
            <div className={`${baseClass}__sig-info`}>
              <DataSet orientation="horizontal" title="Path" value={path} />
              {sigInfo?.hash_sha256 && (
                <DataSet
                  orientation="horizontal"
                  title="Hash"
                  value={sigInfo.hash_sha256}
                />
              )}
            </div>
          );
        })}
    </Card>
  );
};

interface ISoftwareDetailsModalProps {
  hostDisplayName: string;
  software: IHostSoftware;
  onExit: () => void;
  isDeviceUser?: boolean;
}

const SoftwareDetailsContent = ({
  software,
}: Pick<ISoftwareDetailsModalProps, "software">) => {
  const { installed_versions } = software;

  // special case when we dont have installed versions. We can only show the
  // software type atm.
  if (!installed_versions || installed_versions.length === 0) {
    return (
      <div className={`${baseClass}__software-details`}>
        <Card
          className={`${baseClass}__version-details`}
          color="grey"
          borderRadiusSize="medium"
        >
          <div className={`${baseClass}__row`}>
            <DataSet
              title="Type"
              value={formatSoftwareType({ source: software.source })}
            />
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className={`${baseClass}__software-details`}>
      {installed_versions?.map((installedVersion) => {
        return (
          <SoftwareDetailsInfo
            key={installedVersion.version}
            installedVersion={installedVersion}
            source={software.source}
            bundleIdentifier={software.bundle_identifier}
          />
        );
      })}
    </div>
  );
};

const InstallDetailsContent = ({
  hostDisplayName,
  software,
}: {
  hostDisplayName: string;
  software: IHostSoftware;
}) => {
  if (hasHostSoftwareAppLastInstall(software)) {
    return (
      <AppInstallDetails
        command_uuid={software.app_store_app.last_install.command_uuid}
        host_display_name={hostDisplayName}
        software_title={software.name}
        status={software.status || undefined} // FIXME: we have a type mismatch here; as a workaroud this will coerce null to undefined, which in turn defaults to "pending"
      />
    );
  } else if (hasHostSoftwarePackageLastInstall(software)) {
    return (
      <SoftwareInstallDetails
        install_uuid={software.software_package.last_install.install_uuid}
        host_display_name={hostDisplayName}
      />
    );
  }

  // caller should ensure this nevers happen
  return null;
};

const TabsContent = ({
  hostDisplayName,
  software,
}: {
  hostDisplayName: string;
  software: IHostSoftware;
}) => {
  return (
    <TabNav>
      <Tabs>
        <TabList>
          <Tab>
            <TabText>Software details</TabText>
          </Tab>
          <Tab>
            <TabText>Install details</TabText>
          </Tab>
        </TabList>
        <TabPanel>
          <SoftwareDetailsContent software={software} />
        </TabPanel>
        <TabPanel>
          <InstallDetailsContent
            hostDisplayName={hostDisplayName}
            software={software}
          />
        </TabPanel>
      </Tabs>
    </TabNav>
  );
};

const SoftwareDetailsModal = ({
  hostDisplayName,
  software,
  isDeviceUser = false,
  onExit,
}: ISoftwareDetailsModalProps) => {
  // install details will not be shown if Fleet doesn't have them, regardless of this setting
  const hideInstallDetails = isDeviceUser;

  // For scrollable modal
  const [isTopScrolling, setIsTopScrolling] = useState(false);
  const topDivRef = useRef<HTMLDivElement>(null);
  const checkScroll = () => {
    if (topDivRef.current) {
      const isScrolling =
        topDivRef.current.scrollHeight > topDivRef.current.clientHeight;
      setIsTopScrolling(isScrolling);
    }
  };

  const hasLastInstall =
    hasHostSoftwarePackageLastInstall(software) ||
    hasHostSoftwareAppLastInstall(software);

  // For scrollable modal (re-rerun when reopened)
  useEffect(() => {
    checkScroll();
    window.addEventListener("resize", checkScroll);
    return () => window.removeEventListener("resize", checkScroll);
  }, []);

  const renderScrollableContent = () => {
    return (
      <div className={`${baseClass}__content`} ref={topDivRef}>
        {hasLastInstall && !hideInstallDetails ? (
          <TabsContent hostDisplayName={hostDisplayName} software={software} />
        ) : (
          <SoftwareDetailsContent software={software} />
        )}
      </div>
    );
  };
  const renderFooter = () => (
    <ModalFooter
      isTopScrolling={isTopScrolling}
      primaryButtons={
        <Button type="submit" onClick={onExit}>
          Done
        </Button>
      }
    />
  );

  return (
    <Modal
      title={software.name}
      className={baseClass}
      onExit={onExit}
      width="large"
    >
      <>
        {renderScrollableContent()}
        {renderFooter()}
      </>
    </Modal>
  );
};

export default SoftwareDetailsModal;
