import Button from "components/buttons/Button";
import Modal from "components/Modal";
import React from "react";
import { formatDistanceStrict } from "date-fns";

const baseClass = "create-linux-key-modal";

interface ICreateLinuxKeyModal {
  isTriggeringCreateLinuxKey: boolean;
  /** fleetd is already handling an earlier request, so no new pop-up is coming. */
  isEscrowInFlight: boolean;
  /** Seconds until the server accepts a new request if fleetd reports nothing (Retry-After). */
  retryAfterSeconds?: number;
  onExit: () => void;
}

// Whole minutes, rounded up so the end user never retries a moment too early.
const formatRetryWait = (seconds?: number) =>
  seconds
    ? formatDistanceStrict(0, seconds * 1000, {
        unit: "minute",
        roundingMethod: "ceil",
      })
    : "a few minutes";

const CreateLinuxKeyModal = ({
  isTriggeringCreateLinuxKey,
  isEscrowInFlight,
  retryAfterSeconds,
  onExit,
}: ICreateLinuxKeyModal) => {
  const renderCloseCta = () => (
    <div className="modal-cta-wrap">
      <Button type="submit" onClick={onExit} className="save-loading">
        Close
      </Button>
    </div>
  );

  const renderInFlightBody = () => (
    <>
      <p>Fleet is already asking this device for a disk encryption key.</p>
      <p>
        On Ubuntu with TPM-backed disk encryption this happens in the background
        and no pop-up appears. Close this window and select <b>Refetch</b> on
        your <b>My device</b> page in a few minutes.
      </p>
      <p>Otherwise:</p>
      <ul>
        <li>
          If the <b>Enter disk encryption passphrase</b> pop-up is open, enter
          the passphrase used to encrypt your device during setup. The pop-up
          might be behind this window.
        </li>
        <li>
          If you already entered your passphrase, Fleet is finishing up. Close
          this window, wait a couple of minutes, and select <b>Refetch</b> on
          your <b>My device</b> page.
        </li>
        <li>
          If the pop-up closed before you entered your passphrase, wait{" "}
          {formatRetryWait(retryAfterSeconds)}, then select <b>Create key</b>{" "}
          again.
        </li>
      </ul>
      {renderCloseCta()}
    </>
  );

  const renderModalBody = () => (
    <>
      <p>
        On Ubuntu with TPM-backed disk encryption, Fleet backs up your recovery
        key automatically in the background — no further action is needed. The
        yellow <b>Disk Encryption</b> banner will clear within 1 hour.
      </p>
      <p>If a pop-up appears asking for your passphrase, follow these steps:</p>
      <ol>
        <li>
          Wait 30 seconds for the <b>Enter disk encryption passphrase</b> pop-up
          to open.
        </li>
        <li>
          In the pop-up, enter the passphrase used to encrypt your device during
          setup.
        </li>
        <li>
          You&apos;re done. The yellow <b>Disk Encryption</b> banner will go
          away in 1 hour. To remove this banner sooner, wait a couple of minutes
          for Fleet to create a new key. Then, close this window and select{" "}
          <b>Refetch</b> on your <b>My device</b> page.
        </li>
        <li>
          If the banner doesn&apos;t go away after 1 hour, please contact your
          IT admin.
        </li>
      </ol>
      {renderCloseCta()}
    </>
  );
  return (
    <Modal
      title="Create key"
      onExit={onExit}
      onEnter={onExit}
      className={baseClass}
      isLoading={isTriggeringCreateLinuxKey}
    >
      {isEscrowInFlight ? renderInFlightBody() : renderModalBody()}
    </Modal>
  );
};

export default CreateLinuxKeyModal;
