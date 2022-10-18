import React, { useContext } from "react";
import { syntaxHighlight } from "utilities/helpers";

import { AppContext } from "context/app";
import { IVulnerability } from "interfaces/vulnerability";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

const baseClass = "preview-data-modal";

interface IPreviewPayloadModalProps {
  onCancel: () => void;
}
interface IJsonPayload {
  timestamp: string;
  vulnerability: IVulnerability;
}

const PreviewPayloadModal = ({
  onCancel,
}: IPreviewPayloadModalProps): JSX.Element => {
  const { isFreeTier } = useContext(AppContext);

  const json: IJsonPayload = {
    timestamp: "0000-00-00T00:00:00Z",
    vulnerability: {
      cve: "CVE-2014-9471",
      details_link: "https://nvd.nist.gov/vuln/detail/CVE-2014-9471",
      epss_probability: 0.7,
      cvss_score: 5.7,
      cisa_known_exploit: true,
      hosts_affected: [
        {
          id: 1,
          display_name: "macbook-1",
          url: "https://fleet.example.com/hosts/1",
        },
        {
          id: 2,
          display_name: "macbook-2",
          url: "https://fleet.example.com/hosts/2",
        },
      ],
    },
  };

  if (isFreeTier) {
    // Premium only features
    delete json.vulnerability.epss_probability;
    delete json.vulnerability.cvss_score;
    delete json.vulnerability.cisa_known_exploit;
  }

  return (
    <Modal
      title={"Example payload"}
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={`${baseClass}__preview-modal`}>
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
