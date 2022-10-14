import React from "react";

import { IIntegrationType } from "interfaces/integration";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";
import JiraTicketScreenshot from "../../../../../../assets/images/jira-screenshot-400x517@2x.png";
import ZendeskTicketScreenshot from "../../../../../../assets/images/zendesk-screenshot-400x455@2x.png";

const baseClass = "preview-ticket-modal";

interface IPreviewTicketModalProps {
  onCancel: () => void;
  integrationType: IIntegrationType;
}

const PreviewTicketModal = ({
  onCancel,
  integrationType,
}: IPreviewTicketModalProps): JSX.Element => {
  const screenshot =
    integrationType === "jira" ? (
      <img
        src={JiraTicketScreenshot}
        alt="Jira ticket"
        className={`${baseClass}__jira-screenshot`}
      />
    ) : (
      <img
        src={ZendeskTicketScreenshot}
        alt="Zendesk ticket"
        className={`${baseClass}__zendesk-screenshot`}
      />
    );

  return (
    <Modal
      title={"Example ticket"}
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
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
        <div className={`${baseClass}__example`}>{screenshot}</div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default PreviewTicketModal;
