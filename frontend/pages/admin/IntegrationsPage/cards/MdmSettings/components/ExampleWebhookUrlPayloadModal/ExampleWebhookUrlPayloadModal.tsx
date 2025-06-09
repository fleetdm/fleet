import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { syntaxHighlight } from "utilities/helpers";

const baseClass = "example-webhook-url-payload-modal";

const EXAMPLE_PAYLOAD = {
  timestamp: "0000-00-00T00:00:00Z",
  host: {
    id: 1,
    uuid: "1234-5678-9101-1121",
    hardware_serial: "V2RG6Y7VYL",
  },
};

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
        <pre className={`${baseClass}__endpoint-preview`}>
          POST https://organization.com/send-request-here
        </pre>
        <div className={`${baseClass}__webhook-preview`}>
          <pre
            // purposely ignore dangerouslySetInnerHTML for this example payload
            // eslint-disable-next-line react/no-danger
            dangerouslySetInnerHTML={{
              __html: syntaxHighlight(EXAMPLE_PAYLOAD),
            }}
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default ExampleWebhookUrlPayloadModal;
