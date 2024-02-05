import React, { useContext } from "react";
import { syntaxHighlight } from "utilities/helpers";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { AppContext } from "context/app";
import { IPolicyWebhookPreviewPayload } from "interfaces/policy";

const baseClass = "preview-data-modal";

interface IPreviewPayloadModalProps {
  onCancel: () => void;
}

interface IHostPreview {
  id: number;
  display_name: string;
  url: string;
}

interface IPreviewPayload {
  timestamp: string;
  policy: IPolicyWebhookPreviewPayload;
  hosts: IHostPreview[];
}

const PreviewPayloadModal = ({
  onCancel,
}: IPreviewPayloadModalProps): JSX.Element => {
  const { isFreeTier } = useContext(AppContext);

  const json: IPreviewPayload = {
    timestamp: "0000-00-00T00:00:00Z",
    policy: {
      id: 1,
      name: "Is Gatekeeper enabled?",
      query: "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
      description: "Checks if gatekeeper is enabled on macOS devices.",
      author_id: 1,
      author_name: "John",
      author_email: "john@example.com",
      resolution: "Turn on Gatekeeper feature in System Preferences.",
      passing_host_count: 2000,
      failing_host_count: 300,
      critical: false,
    },
    hosts: [
      {
        id: 1,
        display_name: "macbook-1",
        url: "https://fleet.example.com/hosts/1",
      },
      {
        id: 2,
        display_name: "macbbook-2",
        url: "https://fleet.example.com/hosts/2",
      },
    ],
  };
  if (isFreeTier) {
    delete json.policy.critical;
  }

  return (
    <Modal
      title="Example payload"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={`${baseClass}__preview-modal`}>
        <p>
          Want to learn more about how automations in Fleet work?{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/automations"
            text="Check out the Fleet documentation"
            newTab
          />
        </p>
        <div className={`${baseClass}__payload-request-preview`}>
          <pre>POST https://server.com/example</pre>
        </div>
        <div className={`${baseClass}__payload-webhook-preview`}>
          <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(json) }} />
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

export default PreviewPayloadModal;
