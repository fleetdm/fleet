import React from "react";

import Modal from "components/Modal";
import { IWebhookActivities } from "interfaces/webhook";

const baseClass = "activity-feed-automations-modal";

interface IActivityFeedAutomationsModal {
  // optional to facilitate loading config
  automationSettings?: IWebhookActivities;
  onSubmit: () => void;
  onExit: () => void;
}

const ActivityFeedAutomationsModal = ({
  automationSettings,
  onSubmit,
  onExit,
}: IActivityFeedAutomationsModal) => {
  return (
    <Modal
      className={baseClass}
      title="Manage automations"
      width="large"
      onExit={onExit}
    >
      <></>
    </Modal>
  );
};

export default ActivityFeedAutomationsModal;
