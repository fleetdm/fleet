import React from "react";

import { IHostUpcomingActivity } from "interfaces/activity";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { upcomingActivityComponentMap } from "pages/hosts/details/cards/Activity/ActivityConfig";
import { noop } from "lodash";

const baseClass = "cancel-activity-modal";

interface ICancelActivityModalProps {
  activity: IHostUpcomingActivity;
  onExit: () => void;
}

const CancelActivityModal = ({
  activity,
  onExit,
}: ICancelActivityModalProps) => {
  const ActivityItemComponent = upcomingActivityComponentMap[activity.type];

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
          <Button variant="alert">Cancel activity</Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Back
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default CancelActivityModal;
