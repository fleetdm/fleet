import React from "react";

import Button from "components/buttons/Button";
import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import { MdmEnrollmentStatus, isAndroidBYO } from "interfaces/mdm";
import { isAndroid, isIPadOrIPhone } from "interfaces/platform";
import PATHS from "router/paths";
import hostAPI from "services/entities/hosts";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import IpadLockPreview from "../../../../../../assets/images/ipad-lock-preview.png";
import IphoneLockPreview from "../../../../../../assets/images/iphone-lock-preview.png";

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
  hostMdmEnrollmentStatus?: MdmEnrollmentStatus | null;
  onSuccess: () => void;
  onClose: () => void;
}

const LockModal = ({
  id,
  platform,
  hostName,
  hostMdmEnrollmentStatus = null,
  onSuccess,
  onClose,
}: ILockModalProps) => {
  const [lockChecked, setLockChecked] = React.useState(false);
  const [isLocking, setIsLocking] = React.useState(false);

  const isAndroidHost = isAndroid(platform);

  const onLock = async () => {
    setIsLocking(true);
    try {
      await hostAPI.lockHost(id);
      onSuccess();
      notify.success(
        isAndroidHost
          ? "Successfully sent request to lock this host."
          : "Locking host or will lock when it comes online."
      );
    } catch (e) {
      const errorReason = getErrorReason(e);
      notify.error(
        isAndroidHost
          ? errorReason ||
              "Couldn't send request to lock this host. Please try again."
          : errorReason,
        { response: e }
      );
    }
    setIsLocking(false);
  };

  const renderDescription = () => {
    const offboardingText = (
      <p>
        During offboarding, lock a host when it needs to be returned to your
        organization.
      </p>
    );

    if (isAndroidHost) {
      const pinText = (
        <p>
          Fleet will generate an unlock PIN. To view it, select{" "}
          <b>Actions &gt; View unlock PIN</b>.
        </p>
      );
      return (
        <>
          {isAndroidBYO(hostMdmEnrollmentStatus) ? (
            <p>Lock the end user out of the work profile.</p>
          ) : (
            offboardingText
          )}
          {pinText}
        </>
      );
    }

    if (isIPadOrIPhone(platform)) {
      return (
        <>
          {offboardingText}
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

    return (
      <>
        {offboardingText}
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
        <Button onClick={onClose} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default LockModal;
