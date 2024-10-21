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

  const preSonomaBody = (
    <>
      <p className={`${baseClass}__description`}>
        To turn on MDM, Apple Inc. requires you to follow the steps below.
      </p>
      <ol>
        <li>
          Open your Mac&apos;s notification center by selecting the date and
          time in the top right corner of your screen.
        </li>
        <li>
          Select the <b>Device Enrollment</b> notification. This will open{" "}
          <b>System Settings</b>. Select <b>Allow</b>.
        </li>
        <li>
          Enter your password, and select <b>Enroll</b>.
        </li>
        <li>
          Select <b>Done</b> to close this window and select Refetch on your My
          device page to tell your organization that MDM is on.
        </li>
      </ol>
    </>
  );

  const sonomaAndAboveBody = (
    <>
      <p className={`${baseClass}__description`}>
        To turn on MDM, Apple Inc. requires that you install a profile.
      </p>
      <ol>
        <li>
          From the Apple menu in the top left corner of your screen, select{" "}
          <b>System Settings</b> or <b>System Preferences</b>.
        </li>
        <li>
          In the sidebar menu, select <b>Enroll in Remote Management</b>, and
          select <b>Enroll</b>.
        </li>
        <li>
          Enter your password, and select <b>Enroll</b>.
        </li>
        <li>
          Close this window and select <b>Refetch</b> on your My device page to
          tell your organization that MDM is on.
        </li>
      </ol>
    </>
  );

  return (
    <Modal
      title="Turn on MDM"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      width="xlarge"
    >
      <div>
        {isMacOsSonomaOrLater ? sonomaAndAboveBody : preSonomaBody}
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
