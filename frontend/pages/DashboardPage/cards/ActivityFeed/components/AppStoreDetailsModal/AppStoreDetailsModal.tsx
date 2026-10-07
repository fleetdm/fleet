// Used in both Apple App Store (VPP) and Android Play Store details modals

import React from "react";

import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import Editor from "components/Editor";
import Modal from "components/Modal";
import { IActivityDetails } from "interfaces/activity";
import { isAndroid, isMobilePlatform } from "interfaces/platform";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import {
  TargetTitle,
  TargetValue,
} from "../LibrarySoftwareDetailsModal/LibrarySoftwareDetailsModal";

const baseClass = "app-store-details-modal";

// Android configuration arrives as a parsed JSON object (axios); iOS/iPadOS
// arrives as an XML plist string. Normalize to a display string.
const stringifyConfiguration = (
  configuration: IActivityDetails["configuration"]
): string | null => {
  if (!configuration) return null;
  if (typeof configuration === "string") return configuration;
  try {
    return JSON.stringify(configuration, null, 2);
  } catch {
    return null;
  }
};

interface IAppStoreDetailsModalProps {
  details: IActivityDetails;
  onCancel: () => void;
}

const AppStoreDetailsModal = ({
  details,
  onCancel,
}: IAppStoreDetailsModalProps) => {
  const { labels_include_any, labels_exclude_any } = details;
  const isAndroidApp = isAndroid(details.platform || "");
  // macOS activities carry version_name + configuration too, but #53641
  // scopes the versioned-app UI to iOS/iPadOS/Android.
  const isVersionedApp = isMobilePlatform(details.platform || "");
  const configurationDisplay = stringifyConfiguration(details.configuration);

  return (
    <Modal
      title="Details"
      width="large"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={`${baseClass}__modal-content`}>
        <DataSet
          title="Name"
          value={getDisplayedSoftwareName(
            details.software_title,
            details.software_display_name
          )}
        />
        {isVersionedApp && details.version_name && (
          <DataSet title="Version" value={details.version_name} />
        )}
        <DataSet
          title={isAndroidApp ? "Google Play ID" : "App Store ID"}
          value={details.app_store_id}
        />
        <DataSet
          title="Self service"
          value={details.self_service ? "Yes" : "No"}
        />
        <DataSet
          title={
            <TargetTitle
              labelIncludeAny={labels_include_any}
              labelExcludeAny={labels_exclude_any}
            />
          }
          value={
            <TargetValue
              labelIncludeAny={labels_include_any}
              labelExcludeAny={labels_exclude_any}
            />
          }
        />
        {isVersionedApp && configurationDisplay && (
          <Editor
            label="Configuration"
            mode={isAndroidApp ? "json" : "xml"}
            value={configurationDisplay}
            readOnly
          />
        )}
      </div>
      <div className="modal-cta-wrap">
        <Button onClick={onCancel}>Close</Button>
      </div>
    </Modal>
  );
};

export default AppStoreDetailsModal;
