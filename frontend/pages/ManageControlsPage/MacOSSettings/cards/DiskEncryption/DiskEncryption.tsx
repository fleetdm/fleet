import React, { useContext, useState } from "react";

import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";

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
  // TODO: replace default value with enable disk encryption API call
  const [showAggregate, setShowAggregate] = useState(false);

  const { isPremiumTier } = useContext(AppContext);

  const onUpdateDiskEncryption = (value: boolean) => {
    setShowAggregate(value);
  };

  return (
    <div className={baseClass}>
      <h2>Disk encryption</h2>
      {!isPremiumTier ? (
        <PremiumFeatureMessage />
      ) : (
        <>
          {showAggregate ? <div>Aggregate</div> : null}
          <Checkbox onChange={onUpdateDiskEncryption} value={showAggregate}>
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
          <Button onClick={onUpdateDiskEncryption}>Save</Button>
        </>
      )}
    </div>
  );
};

export default DiskEncryption;
