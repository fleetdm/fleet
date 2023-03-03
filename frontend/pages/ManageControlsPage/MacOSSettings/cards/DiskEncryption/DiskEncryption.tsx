import React, { useContext, useState } from "react";

import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";

import DiskEncryptionTable from "./components/DiskEncryptionTable";

const baseClass = "disk-encryption";

const PremiumFeatureMessage = () => {
  return (
    <div className={`${baseClass}__premium-feature-message`}>
      <p>
        This feature is included in Fleet Premium.{" "}
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/upgrade"
          newTab
        />
      </p>
    </div>
  );
};

const DiskEncryption = () => {
  const data = {
    applied: 1,
    action_required: 10,
    enforcing: 1000,
    failed: 10000,
    removing_enforcement: 100000,
  };

  // TODO: replace default value with enable disk encryption API call
  const [showAggregate, setShowAggregate] = useState(false);
  const [isChecked, setIsChecked] = useState(false);

  const { isPremiumTier } = useContext(AppContext);

  const onToggleCheckbox = (value: boolean) => {
    setIsChecked(value);
  };

  const onUpdateDiskEncryption = () => {
    setShowAggregate(isChecked);
  };

  return (
    <div className={baseClass}>
      <h2>Disk encryption</h2>
      {!isPremiumTier ? (
        <PremiumFeatureMessage />
      ) : (
        <>
          {showAggregate ? <DiskEncryptionTable aggregateData={data} /> : null}
          <Checkbox
            onChange={onToggleCheckbox}
            value={isChecked}
            className={`${baseClass}__checkbox`}
          >
            On
          </Checkbox>
          <p>
            Apple calls this “FileVault.” If turned on, hosts’ disk encryption
            keys will be stored in Fleet.{" "}
            <CustomLink
              text="Learn more"
              url="https://fleetdm.com/docs/controls#disk-encryption"
              newTab
            />
          </p>
          <Button
            className={`${baseClass}__save-button`}
            onClick={onUpdateDiskEncryption}
          >
            Save
          </Button>
        </>
      )}
    </div>
  );
};

export default DiskEncryption;
