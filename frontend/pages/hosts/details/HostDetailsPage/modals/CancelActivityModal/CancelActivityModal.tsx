import React, { useContext } from "react";
import { noop } from "lodash";

import { IHostUpcomingActivity } from "interfaces/activity";
import activitiesAPI from "services/entities/activities";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import { upcomingActivityComponentMap } from "pages/hosts/details/cards/Activity/ActivityConfig";

import { getErrorMessage } from "./helpers";

const baseClass = "cancel-activity-modal";

interface ICancelActivityModalProps {
  hostId: number;
  activity: IHostUpcomingActivity;
  onCancel: () => void;
}

const CancelActivityModal = ({
  hostId,
  activity,
  onCancel,
}: ICancelActivityModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const ActivityItemComponent = upcomingActivityComponentMap[activity.type];

  const onCancelActivity = async () => {
    try {
      await activitiesAPI.cancelHostActivity(hostId, activity.uuid);
      renderFlash("success", "Activity successfully canceled.");
    } catch (error) {
      // TODO: hook up error message when API is updated
      renderFlash("error", getErrorMessage(error));
    }
    onCancel();
  };

  return (
    <Modal className={baseClass} title="Cancel activity" onExit={onCancel}>
      <>
        <div className={`${baseClass}__content`}>
          <p>
            If the activity is happening on the host it will still complete.
            Results won&apos;t appear in Fleet.
          </p>
          <ActivityItemComponent
            tab="upcoming"
            activity={activity}
            onCancel={noop}
            onShowDetails={noop}
            isSoloActivity
          />
        </div>
        <div className="modal-cta-wrap">
          <Button variant="alert" onClick={onCancelActivity}>
            Cancel activity
          </Button>
          <Button variant="inverse-alert" onClick={onCancel}>
            Back
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default CancelActivityModal;
