import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Textarea from "components/Textarea";
import { IActivityDetails } from "interfaces/activity";

const baseClass = "activity-automation-details-modal";

interface IActivityAutomationDetailsModalProps {
  details: IActivityDetails;
  onCancel: () => void;
}

const ActivityAutomationDetailsModal = ({
  details,
  onCancel,
}: IActivityAutomationDetailsModalProps) => {
  const renderContent = () => {
    return (
      <>
        <div className={`${baseClass}__modal-content`}>
          <p>
            Fleet will send a JSON payload to this URL whenever a new activity
            is generated:
          </p>
          <div className={`${baseClass}__webhook-url`}>
            <Textarea className={`${baseClass}__webhook-url-textarea`}>
              {details.webhook_url}
            </Textarea>
          </div>
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal
      title="Details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {renderContent()}
    </Modal>
  );
};

export default ActivityAutomationDetailsModal;
