import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IDeviceUserResponse } from "interfaces/host";

interface IAutoEnrollMdmModalProps {
  host: IDeviceUserResponse["host"];
  onCancel: () => void;
}

const baseClass = "auto-enroll-mdm-modal";

const AutoEnrollMdmModal = ({
  host: { platform, os_version },
  onCancel,
}: IAutoEnrollMdmModalProps): JSX.Element => {
  let isMacOsSonomaOrLater = false;
  if (platform === "darwin" && os_version.startsWith("macOS ")) {
    const [major] = os_version
      .replace("macOS ", "")
      .split(".")
      .map((s) => parseInt(s, 10));
    isMacOsSonomaOrLater = major >= 14;
  }

  return (
    <Modal
      title="Turn on MDM"
      onExit={onCancel}
      className={baseClass}
      width="xlarge"
    >
      <div>
        <p className={`${baseClass}__description`}>
          To turn on MDM, Apple Inc. requires that you install a profile.
        </p>
        <ol>
          <li>
            From the Apple menu in the top left corner of your screen, select{" "}
            <b>System Settings</b>.
          </li>
          <li>
            {isMacOsSonomaOrLater ? (
              <>
                In the sidebar menu, select <b>Enroll in Remote Management</b>,
                and select <b>Enroll</b>.
              </>
            ) : (
              <>
                In the search bar, type “Profiles.” Select <b>Profiles</b>, find
                and double-click the <b>[Organization name] enrollment</b>{" "}
                profile.
              </>
            )}
          </li>
          <li>
            Enter your password, and select <b>Enroll</b>.
          </li>
          <li>
            Select <b>Done</b> to close this window and select <b>Refetch</b> on
            your My Device page to tell your organization that MDM is on.
          </li>
        </ol>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default AutoEnrollMdmModal;
