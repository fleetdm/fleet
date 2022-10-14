import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";
import JiraTicket from "../../../../../../assets/images/ticket-policies-jira-screenshot-400x419@2x.png";
import ZendeskTicket from "../../../../../../assets/images/ticket-policies-zendesk-screenshot-400x515@2x.png";

const baseClass = "preview-ticket-modal";

interface IPreviewTicketModalProps {
  type?: "jira" | "zendesk";
  onCancel: () => void;
}

const PreviewTicketModal = ({
  type,
  onCancel,
}: IPreviewTicketModalProps): JSX.Element => {
  return (
    <Modal title={"Example ticket"} onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}`}>
        <p>
          Want to learn more about how automations in Fleet work?{" "}
          <a
            href="https://fleetdm.com/docs/using-fleet/automations"
            target="_blank"
            rel="noopener noreferrer"
          >
            Check out the Fleet documentation
            <img src={ExternalLinkIcon} alt="Open external link" />
          </a>
        </p>
        <div className={`${baseClass}__example`}>
          <img
            className={`${baseClass}__screenshot`}
            alt="Example policies automation ticket"
            src={type === "zendesk" ? ZendeskTicket : JiraTicket}
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PreviewTicketModal;
