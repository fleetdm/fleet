import React, { Dispatch, SetStateAction } from "react";
import classnames from "classnames";

import Checkbox from "components/forms/fields/Checkbox";

import { ISoftwareVppFormData } from "pages/SoftwarePage/SoftwareAddPage/SoftwareAppStoreVpp/SoftwareVppForm/SoftwareVppForm";

const baseClass = "software-options-selector";

interface ISoftwareOptionsSelector {
  platform: string;
  className?: string;
  formData: ISoftwareVppFormData;
  setFormData: Dispatch<SetStateAction<ISoftwareVppFormData>>;
}

const SoftwareOptionsSelector = ({
  platform,
  className,
  formData,
  setFormData,
}: ISoftwareOptionsSelector) => {
  const classNames = classnames(baseClass, className);

  const isSelfServiceDisabled = platform === "ios" || platform === "ipados";
  const isAutomaticInstallDisabled =
    platform === "ios" || platform === "ipados";

  return (
    <div className={"form-field"}>
      <div className="form-field__label">Options</div>
      {isSelfServiceDisabled && (
        <p>
          Currently, self-service and automatic installation are not available
          for iOS and iPadOS. Manually install on the <b>Host details</b> page
          for each host.
        </p>
      )}
      <Checkbox
        value={formData.selfService}
        onChange={(newVal: boolean) =>
          setFormData({ ...formData, selfService: newVal })
        }
        className={`${baseClass}__self-service-checkbox`}
        tooltipContent={
          !isSelfServiceDisabled && (
            <>
              End users can install from <b>Fleet Desktop</b> {">"}{" "}
              <b>Self-service</b>.
            </>
          )
        }
        disabled={isSelfServiceDisabled}
      >
        Self-service
      </Checkbox>
      <Checkbox
        value={formData.automaticInstall}
        onChange={(newVal: boolean) =>
          setFormData({ ...formData, automaticInstall: newVal })
        }
        className={`${baseClass}__automatic-install-checkbox`}
        tooltipContent={
          !isAutomaticInstallDisabled && (
            <>Automatically install only on hosts missing this software.</>
          )
        }
        disabled={isAutomaticInstallDisabled}
      >
        Automatic install
      </Checkbox>
    </div>
  );
};

export default SoftwareOptionsSelector;
