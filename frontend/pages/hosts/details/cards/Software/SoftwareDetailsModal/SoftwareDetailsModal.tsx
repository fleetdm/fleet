import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import {
  IHostSoftware,
  ISoftwareInstallVersion,
  formatSoftwareType,
} from "interfaces/software";

import Modal from "components/Modal";
import TabsWrapper from "components/TabsWrapper";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import { dateAgo } from "utilities/date_format";

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
  source: string;
  bundleIdentifier: string;
}

const SoftwareDetailsInfo = ({
  installedVersion,
  source,
  bundleIdentifier,
}: ISoftwareDetailsInfoProps) => {
  return (
    <div className={`${baseClass}__details-info`}>
      <div className={`${baseClass}__row`}>
        <DataSet title="Version" value={installedVersion.version} />
        <DataSet title="Type" value={formatSoftwareType({ source })} />
        <DataSet title="Bundle identifier" value={bundleIdentifier} />
        {installedVersion.last_opened_at && (
          <DataSet
            title="Last used"
            value={dateAgo(installedVersion.last_opened_at)}
          />
        )}
      </div>
      <div className={`${baseClass}__row`}>
        <DataSet
          title="File path"
          value={installedVersion.installed_paths.map((path) => (
            <>{path}</>
          ))}
        />
      </div>
      <div className={`${baseClass}__row`}>
        <DataSet
          title="Vulnerabilities"
          value={generateVulnerabilitiesValue(installedVersion.vulnerabilities)}
        />
      </div>
    </div>
  );
};

interface ISoftwareDetailsModalProps {
  software: IHostSoftware;
  onExit: () => void;
}

const SoftwareDetailsModal = ({
  software,
  onExit,
}: ISoftwareDetailsModalProps) => {
  const renderSoftwareDetails = () => {
    if (
      !software.installed_versions ||
      software.installed_versions.length === 0
    ) {
      return null;
    }

    return software.installed_versions.map((installedVersion) => {
      return (
        <SoftwareDetailsInfo
          key={installedVersion.version}
          installedVersion={installedVersion}
          source={software.source}
          bundleIdentifier={software.bundle_identifier}
        />
      );
    });
  };

  return (
    <Modal title={software.name} className={baseClass} onExit={onExit}>
      <>
        <TabsWrapper>
          <Tabs>
            <TabList>
              <Tab>Software details</Tab>
              <Tab>Install Details</Tab>
            </TabList>
            <TabPanel>{renderSoftwareDetails()}</TabPanel>
            <TabPanel>test 2</TabPanel>
          </Tabs>
        </TabsWrapper>
        <div className="modal-cta-wrap">
          <Button type="submit" variant="brand" onClick={onExit}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default SoftwareDetailsModal;
