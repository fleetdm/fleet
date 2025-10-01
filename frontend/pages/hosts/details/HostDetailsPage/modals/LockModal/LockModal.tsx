import React, { useContext } from "react";
import { Link } from "react-router";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import hostAPI from "services/entities/hosts";
import { isIPadOrIPhone } from "interfaces/platform";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import Card from "components/Card";

import IphoneLockPreview from "../../../../../../../assets/images/iphone.png";

const baseClass = "lock-modal";

const IosOrIpadLockPreview = () => {
  return (
    <Card className={`${baseClass}__ios-ipad-lock-preview`}>
      <h3>End user experience</h3>
      <p>
        Instead of &quot;Fleet&quot;, the message will show the{" "}
        <b>Organization Name</b> that you configured in{" "}
        <Link to={PATHS.ADMIN_ORGANIZATION_INFO}>Organization settings</Link>.
      </p>
      <img src={IphoneLockPreview} alt="iPhone with a lock screen message" />
    </Card>
  );
};

interface ILockModalProps {
  id: number;
  platform: string;
  hostName: string;
  onSuccess: () => void;
  onClose: () => void;
}

const LockModal = ({
  id,
  platform,
  hostName,
  onSuccess,
  onClose,
}: ILockModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [lockChecked, setLockChecked] = React.useState(false);
  const [isLocking, setIsLocking] = React.useState(false);

  const onLock = async () => {
    setIsLocking(true);
    try {
      await hostAPI.lockHost(id);
      onSuccess();
      renderFlash("success", "Locking host or will lock when it comes online.");
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    onClose();
    setIsLocking(false);
  };

  const renderDescription = () => {
    if (isIPadOrIPhone(platform)) {
      // if (true) {
      return (
        <p>
          This will enable{" "}
          <CustomLink
            url="https://fleetdm.com/learn-more-about/managed-lost-mode"
            newTab
            text="Lost Mode"
          />
          . It can only be unlocked through Fleet.
        </p>
      );
    }

    return (
      <>
        <p>Lock a host when it needs to be returned to your organization.</p>
        {platform === "darwin" && (
          <p>Fleet will generate a six-digit unlock PIN.</p>
        )}
      </>
    );
  };

  return (
    <Modal className={baseClass} title="Lock host" onExit={onClose}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <div className={`${baseClass}__description`}>
            {renderDescription()}
          </div>
          <div className={`${baseClass}__confirm-message`}>
            <span>
              <b>Please check to confirm:</b>
            </span>
            <Checkbox
              wrapperClassName={`${baseClass}__lock-checkbox`}
              value={lockChecked}
              onChange={(value: boolean) => setLockChecked(value)}
            >
              I wish to lock <b>{hostName}</b>
            </Checkbox>
          </div>
        </div>
        {isIPadOrIPhone(platform) && <IosOrIpadLockPreview />}
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onLock}
            className="delete-loading"
            disabled={!lockChecked}
            isLoading={isLocking}
          >
            Done
          </Button>
          <Button onClick={onClose} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default LockModal;
