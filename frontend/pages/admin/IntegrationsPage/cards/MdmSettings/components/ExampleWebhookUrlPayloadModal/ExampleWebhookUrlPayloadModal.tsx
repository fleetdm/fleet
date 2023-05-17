import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "example-webhook-url-payload-modal";

interface IExampleWebhookUrlPayloadModalProps {
  onCancel: () => void;
}

const ExampleWebhookUrlPayloadModal = ({
  onCancel,
}: IExampleWebhookUrlPayloadModalProps) => {
  return (
    <Modal title="Example payload" onExit={onCancel} className={baseClass}>
      <>
        <p>
          An example request sent to your configured <b>Webhook URL</b>.
        </p>
        <div className={`${baseClass}__host-status-webhook-preview`}>
          <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(json) }} />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default ExampleWebhookUrlPayloadModal;
