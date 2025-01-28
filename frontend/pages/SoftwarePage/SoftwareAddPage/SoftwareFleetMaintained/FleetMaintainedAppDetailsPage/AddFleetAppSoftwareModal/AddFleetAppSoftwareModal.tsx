import React from "react";
import { noop } from "lodash";

import Modal from "components/Modal";
import Spinner from "components/Spinner";

const baseClass = "add-fleet-app-software-modal";

const AddFleetAppSoftwareModal = () => {
  return (
    <Modal
      className={baseClass}
      title="Add software"
      width="large"
      onExit={noop}
      disableClosingModal
    >
      <>
        <Spinner centered={false} className={`${baseClass}__spinner`} />
        <p>
          Uploading software so that it&apos;s available for install. This may
          take a few minutes.
        </p>
      </>
    </Modal>
  );
};

export default AddFleetAppSoftwareModal;
