import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

const baseClass = "debug-logging-modal";

interface IDurationOption {
  label: string;
  value: string; // Go-duration string
}

const DURATION_OPTIONS: IDurationOption[] = [
  { label: "15 minutes", value: "15m" },
  { label: "1 hour", value: "1h" },
  { label: "4 hours", value: "4h" },
  { label: "24 hours (default)", value: "24h" },
  { label: "7 days (max)", value: "168h" },
];

const DEFAULT_DURATION = "24h";

interface IDebugLoggingModalProps {
  hostId: number;
  hostName: string;
  isCurrentlyActive: boolean;
  onSuccess: (orbitDebugUntil: string | null) => void;
  onClose: () => void;
}

/**
 * Modal for enabling (or disabling) orbit debug logging on a single host.
 *
 * Enabling picks a duration (server caps at 7d; default 24h). Disabling
 * clears any active host-level override and reverts the host to the team
 * default. Orbit picks up the change on its next config poll (up to 30s)
 * without restarting.
 *
 * See docs/Contributing/architecture/orbit-debug-logging.md.
 */
const DebugLoggingModal = ({
  hostId,
  hostName,
  isCurrentlyActive,
  onSuccess,
  onClose,
}: IDebugLoggingModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [duration, setDuration] = useState<string>(DEFAULT_DURATION);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const title = isCurrentlyActive
    ? "Disable debug logging"
    : "Enable debug logging";

  const ctaLabel = isCurrentlyActive ? "Disable" : "Enable";

  const onSubmit = async () => {
    setIsSubmitting(true);
    try {
      const response = await hostAPI.setOrbitDebugLogging(
        hostId,
        !isCurrentlyActive,
        isCurrentlyActive ? undefined : duration
      );
      onSuccess(response.orbit_debug_until);
      renderFlash(
        "success",
        isCurrentlyActive
          ? `Debug logging disabled on ${hostName}.`
          : `Debug logging enabled on ${hostName}.`
      );
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    setIsSubmitting(false);
  };

  return (
    <Modal className={baseClass} title={title} onExit={onClose}>
      <div className={`${baseClass}__modal-content`}>
        {isCurrentlyActive ? (
          <p>
            Turn off orbit debug logging on <b>{hostName}</b>. Orbit will revert
            to the team-level setting on its next check-in (up to 30 seconds).
          </p>
        ) : (
          <>
            <p>
              Turn on orbit debug logging for <b>{hostName}</b>. Orbit and
              osqueryd will produce verbose logs until the duration elapses. The
              change applies on the next check-in (up to 30 seconds) with no
              restart required.
            </p>
            <Dropdown
              label="Duration"
              name="debug-logging-duration"
              options={DURATION_OPTIONS}
              value={duration}
              onChange={(value: string) => setDuration(value)}
              searchable={false}
              wrapperClassName={`${baseClass}__duration-dropdown`}
            />
          </>
        )}
      </div>
      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onSubmit}
          className="transfer-loading"
          variant={isCurrentlyActive ? "alert" : undefined}
          isLoading={isSubmitting}
        >
          {ctaLabel}
        </Button>
        <Button onClick={onClose} variant="inverse">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DebugLoggingModal;
