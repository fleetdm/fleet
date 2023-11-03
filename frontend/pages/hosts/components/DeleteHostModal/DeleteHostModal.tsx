import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

const baseClass = "delete-host-modal";

interface IDeleteHostModalProps {
  onSubmit: () => void;
  onCancel: () => void;
  /** Manage host page only */
  isAllMatchingHostsSelected?: boolean;
  /** Manage host page only */
  selectedHostIds?: number[];
  /** Manage host page only */
  hostsCount?: number;
  /** Host details page only */
  hostName?: string;
  isUpdating: boolean;
}

const DeleteHostModal = ({
  onSubmit,
  onCancel,
  isAllMatchingHostsSelected,
  selectedHostIds,
  hostsCount,
  hostName,
  isUpdating,
}: IDeleteHostModalProps): JSX.Element => {
  const hostText = () => {
    if (selectedHostIds) {
      return `${selectedHostIds.length}${
        isAllMatchingHostsSelected ? "+" : ""
      } ${selectedHostIds.length === 1 ? "host" : "hosts"}`;
    }
    return hostName;
  };
  const largeVolumeText = (): string => {
    if (selectedHostIds && isAllMatchingHostsSelected && hostsCount && hostsCount >= 500) {
      return " When deleting a large volume of hosts, it may take some time for this change to be reflected in the UI."
    }
    return ""
  }

  return (
    <Modal
      title={"Delete host"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <form className={`${baseClass}__form`}>
        <p>
          This action will delete <b>{hostText()}</b> from your Fleet instance.{largeVolumeText()}
        </p>
        <p>If the hosts come back online, they will automatically re-enroll.</p>
        <p>
          To prevent re-enrollment,{" "}
          <CustomLink
            url={
              "https://fleetdm.com/docs/using-fleet/faq#how-can-i-uninstall-the-osquery-agent"
            }
            text={"uninstall the osquery agent"}
            newTab
          />
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            isLoading={isUpdating}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default DeleteHostModal;
