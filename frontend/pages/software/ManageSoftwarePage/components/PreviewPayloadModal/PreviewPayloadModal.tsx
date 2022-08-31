import React from "react";
import { syntaxHighlight } from "utilities/helpers";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

const baseClass = "preview-data-modal";

interface IPreviewPayloadModalProps {
  onCancel: () => void;
}

const PreviewPayloadModal = ({
  onCancel,
}: IPreviewPayloadModalProps): JSX.Element => {
  const json = {
    timestamp: "0000-00-00T00:00:00Z",
    vulnerability: {
      cve: "CVE-2014-9471",
      details_link: "https://nvd.nist.gov/vuln/detail/CVE-2014-9471",
      hosts_affected: [
        {
          id: 1,
          hostname: "macbook-1",
          url: "https://fleet.example.com/hosts/1",
        },
        {
          id: 2,
          hostname: "macbook-2",
          url: "https://fleet.example.com/hosts/2",
        },
      ],
    },
  };

  return (
    <Modal title={"Example payload"} onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__preview-modal`}>
        <p>
          Want to learn more about how automations in Fleet work?{" "}
          <a
            href="https://fleetdm.com/docs/using-fleet/automations"
            target="_blank"
            rel="noopener noreferrer"
          >
            Check out the Fleet documentation&nbsp;
            <FleetIcon name="external-link" />
          </a>
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
