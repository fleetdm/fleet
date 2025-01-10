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
  onExit: () => void;
}

const CancelActivityModal = ({
  hostId,
  activity,
  onExit,
}: ICancelActivityModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const ActivityItemComponent = upcomingActivityComponentMap[activity.type];

  const onCancelActivity = async () => {
    try {
      await activitiesAPI.cancelActivity(hostId, activity.uuid);
      renderFlash("success", "Activity successfully canceled.");
    } catch (error) {
      // TODO: hook up error message when API is updated
      renderFlash("error", getErrorMessage(error));
    }
    onExit();
  };

  return (
    <Modal className={baseClass} title="Cancel activity" onExit={onExit}>
      <>
        <ActivityItemComponent
          tab="upcoming"
          activity={activity}
          onCancel={noop}
          onShowDetails={noop}
          soloActivity
        />
        <div className="modal-cta-wrap">
          <Button variant="alert" onClick={onCancelActivity}>
            Cancel activity
          </Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Back
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default CancelActivityModal;
