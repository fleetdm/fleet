import React, { useContext } from "react";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import hostAPI from "services/entities/hosts";
import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import Card from "components/Card";

import IphoneLockPreview from "../../../../../../../assets/images/iphone-lock-preview.png";
import IpadLockPreview from "../../../../../../../assets/images/ipad-lock-preview.png";

const baseClass = "lock-modal";

const IosOrIpadLockPreview = ({ platform }: { platform: string }) => {
  const isIPad = platform === "ipados";
  const previewImage = isIPad ? IpadLockPreview : IphoneLockPreview;
  const altText = isIPad
    ? "iPad with a lock screen message"
    : "iPhone with a lock screen message";

  return (
    <Card
      color="grey"
      paddingSize="xlarge"
      className={`${baseClass}__ios-ipad-lock-preview`}
    >
      <h3>End user experience</h3>
      <p>
        Instead of &quot;Fleet&quot;, the message will show the{" "}
        <b>Organization name</b> that you configured in{" "}
        <CustomLink
          url={PATHS.ADMIN_ORGANIZATION_INFO}
          text="Organization settings"
        />
        .
      </p>
      <img src={previewImage} alt={altText} />
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

  const isAndroidHost = isAndroid(platform);

  const onLock = async () => {
    setIsLocking(true);
    try {
      await hostAPI.lockHost(id);
      onSuccess();
      // Android uses the Figma-specified copy; other platforms keep their existing message to
      // avoid regressing copy they were QA'd against.
      renderFlash(
        "success",
        isAndroidHost
          ? "Successfully sent request to lock this host."
          : "Locking host or will lock when it comes online."
      );
    } catch (e) {
      // Android: surface the backend error reason when available (e.g. "Android MDM isn't turned
      // on.", "Host has pending lock request.") and fall back to the Figma copy when the error
      // has no extractable reason. Other platforms keep their existing getErrorReason behavior.
      const errorReason = getErrorReason(e);
      renderFlash(
        "error",
        isAndroidHost
          ? errorReason ||
              "Couldn't send request to lock this host. Please try again."
          : errorReason
      );
    }
    setIsLocking(false);
  };

  const renderDescription = () => {
    if (isIPadOrIPhone(platform)) {
      return (
        <>
          <p>
            This enables what Apple calls{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/managed-lost-mode"
              newTab
              text="Lost Mode"
            />
            .
          </p>
          <p> It can only be unlocked through Fleet.</p>
          <p>
            If the host is turned off or restarted while locked, it will
            disconnect from Wi-Fi, and you won&apos;t be able to unlock it
            remotely.{" "}
            <CustomLink
              newTab
              text="Learn more"
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/unlock-ios-ipados`}
            />
          </p>
        </>
      );
    }

    if (isAndroid(platform)) {
      // Per Figma: AMAPI's LOCK targets the work profile on BYO and the device on COBO. There is
      // no Fleet-side unlock — the user unlocks via their existing password/PIN. Copy is the same
      // for both ownership modes per the Figma dev note.
      return (
        <p>
          Locking will enforce the host lock screen and require the user to
          enter their password/PIN to regain access.
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
    <Modal className={baseClass} title="Lock" onExit={onClose}>
      <div className={`${baseClass}__modal-content`}>
        <div className={`${baseClass}__description`}>{renderDescription()}</div>
        <div className={`${baseClass}__confirm-message`}>
          <span>
            <b>Confirm:</b>
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
      {isIPadOrIPhone(platform) && <IosOrIpadLockPreview platform={platform} />}
      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onLock}
          className="delete-loading"
          disabled={!lockChecked}
          isLoading={isLocking}
        >
          Lock
        </Button>
        <Button onClick={onClose} variant="inverse">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default LockModal;
