import React from "react";

import { InstallType } from "interfaces/software";
import { ILabelIdentifier } from "interfaces/label";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FleetAce from "components/FleetAce";
import CustomLink from "components/CustomLink";
import Editor from "components/Editor";

const baseClass = "package-options-modal";

interface IPackageOptionsModal {
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  onExit: () => void;
  installType?: InstallType;
  labels?: ILabelIdentifier[];
  labelsIncludeAny?: boolean;
}

const PackageOptionsModal = ({
  installScript,
  preInstallQuery,
  postInstallScript,
  onExit,
  installType,
  labels,
  labelsIncludeAny,
}: IPackageOptionsModal) => {
  return (
    <Modal className={baseClass} title="Options" onExit={onExit}>
      <>
        <p className={`${baseClass}__header`}>
          Options are read-only. To change options, delete software and add
          again.
        </p>
        <div className="form">
          <>
            {installType && (
              <div className="form-field">
                <div className="form-field__label">Install</div>
                {installType[0].toUpperCase() + installType.slice(1)}
              </div>
            )}
            {!!labels?.length && (
              <div className="form-field">
                <div className="form-field__label">Target</div>
                <div className="fleet-labels-list-header">
                  Software will only be available for install on hosts that{" "}
                  {!labelsIncludeAny && "don't "}have <b>any</b> of these
                  labels:
                </div>
                <div className={`${baseClass}__fleet-labels-list`}>
                  {labels.map((label) => (
                    <div key={label.id} className={`${baseClass}__fleet-label`}>
                      {label.name}
                    </div>
                  ))}
                </div>
              </div>
            )}
            {preInstallQuery && (
              <FleetAce
                readOnly
                className="form-field"
                value={preInstallQuery}
                label="Pre-install condition"
                name="preInstallQuery"
                maxLines={10}
                helpText={
                  <>
                    Software will be installed only if the{" "}
                    <CustomLink
                      className={`${baseClass}__table-link`}
                      text="query returns results"
                      url="https://fleetdm.com/tables"
                      newTab
                    />
                  </>
                }
              />
            )}
            <Editor
              readOnly
              wrapEnabled
              maxLines={10}
              name="install-script"
              value={installScript}
              helpText="Fleet will run this script on hosts to install software."
              label="Install script"
              isFormField
            />
            {postInstallScript && (
              <Editor
                readOnly
                wrapEnabled
                label="Post-install script"
                className="form-field"
                name="post-install-script-editor"
                maxLines={10}
                value={postInstallScript}
                helpText="Fleet will run this script after install."
                isFormField
              />
            )}
          </>
        </div>
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onExit}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default PackageOptionsModal;
