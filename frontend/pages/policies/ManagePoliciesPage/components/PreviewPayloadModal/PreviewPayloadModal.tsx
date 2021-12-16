/* This component is used for creating and editing both global and team scheduled queries */

import React from "react";
import { syntaxHighlight } from "fleet/helpers";

import ReactTooltip from "react-tooltip";
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
  const post = "POST https://server.com/example";

  const json = {
    action: "snapshot",
    snapshot: [
      {
        remote_address: "0.0.0.0",
        remote_port: "0",
        cmdline: "/usr/sbin/syslogd",
      },
    ],
    name: "xxxxxxx",
    hostIdentifier: "xxxxxxx",
    calendarTime: "xxx xxx  x xx:xx:xx xxxx UTC",
    unixTime: "xxxxxxxxx",
    epoch: "xxxxxxxxx",
    counter: "x",
    numerics: "x",
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
          <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(post) }} />
        </div>
        <div className={`${baseClass}__payload-webhook-preview`}>
          <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(json) }} />
        </div>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="brand"
          >
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PreviewPayloadModal;
