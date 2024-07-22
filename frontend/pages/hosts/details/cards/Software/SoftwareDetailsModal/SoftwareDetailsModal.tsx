import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import {
  IHostSoftware,
  IHostSoftwareWithLastInstall,
  ISoftwareInstallVersion,
  formatSoftwareType,
  hasLastInstall,
} from "interfaces/software";

import Modal from "components/Modal";
import TabsWrapper from "components/TabsWrapper";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import { dateAgo } from "utilities/date_format";

import { AppInstallDetails } from "components/ActivityDetails/InstallDetails/AppInstallDetails";
import { SoftwareInstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareInstallDetails";
import TooltipTruncatedText from "components/TooltipTruncatedText";

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
  bundleIdentifier?: string;
}

const SoftwareDetailsInfo = ({
  installedVersion,
  source,
  bundleIdentifier,
}: ISoftwareDetailsInfoProps) => {
  const { vulnerabilities, installed_paths } = installedVersion;

  return (
    <div className={`${baseClass}__details-info`}>
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
      {!!installed_paths?.length && (
        <div className={`${baseClass}__row`}>
          <DataSet
            className={`${baseClass}__file-path-data-set`}
            title="File path"
            value={
              <div className={`${baseClass}__file-path-values`}>
                {installed_paths.map((path) => (
                  <TooltipTruncatedText value={path} />
                ))}
              </div>
            }
          />
        </div>
      )}
      {vulnerabilities && vulnerabilities.length !== 0 && (
        <div className={`${baseClass}__row`}>
          <DataSet
            title="Vulnerabilities"
            value={generateVulnerabilitiesValue(vulnerabilities)}
          />
        </div>
      )}
    </div>
  );
};

interface ISoftwareDetailsModalProps {
  hostDisplayName: string;
  software: IHostSoftware;
  onExit: () => void;
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
        <DataSet
          title="Type"
          value={formatSoftwareType({ source: software.source })}
        />
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

const TabsContent = ({
  hostDisplayName,
  software,
}: {
  hostDisplayName: string;
  software: IHostSoftwareWithLastInstall;
}) => {
  return (
    <TabsWrapper>
      <Tabs>
        <TabList>
          <Tab>Software details</Tab>
          <Tab>Install details</Tab>
        </TabList>
        <TabPanel>
          <SoftwareDetailsContent software={software} />
        </TabPanel>
        <TabPanel>
          {software.app_store_app ? (
            <AppInstallDetails
              command_uuid={software.last_install.install_uuid}
              host_display_name={hostDisplayName}
              software_title={software.name}
              status={
                software.status || undefined // FIXME: we have a type mismatch here; as a workaroud this will coerce null to undefined, which in turn defaults to "pending"
              }
            />
          ) : (
            <SoftwareInstallDetails
              installUuid={software.last_install.install_uuid || ""}
            />
          )}
        </TabPanel>
      </Tabs>
    </TabsWrapper>
  );
};

const SoftwareDetailsModal = ({
  hostDisplayName,
  software,
  onExit,
}: ISoftwareDetailsModalProps) => {
  return (
    <Modal title={software.name} className={baseClass} onExit={onExit}>
      <>
        {!hasLastInstall(software) ? (
          <SoftwareDetailsContent software={software} />
        ) : (
          <TabsContent hostDisplayName={hostDisplayName} software={software} />
        )}
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
