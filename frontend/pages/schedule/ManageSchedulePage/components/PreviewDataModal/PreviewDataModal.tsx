/* This component is used for creating and editing both global and team scheduled queries */

import React from "react";
import { syntaxHighlight } from "fleet/helpers";

import ReactTooltip from "react-tooltip";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

import QuestionIcon from "../../../../../../assets/images/icon-question-16x16@2x.png";

const baseClass = "preview-data-modal";

interface IPreviewDataModalProps {
  onCancel: () => void;
}

const PreviewDataModal = ({
  onCancel,
}: IPreviewDataModalProps): JSX.Element => {
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
    <Modal title={"Example data"} onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__preview-modal`}>
        <p>
          The data sent to your configured log destination will look similar to
          the following JSON:{" "}
          <span
            className={`tooltip__tooltip-icon`}
            data-tip
            data-for={"preview-tooltip"}
            data-tip-disable={false}
          >
            <img alt="preview schedule" src={QuestionIcon} />
          </span>
          <ReactTooltip
            place="bottom"
            type="dark"
            effect="solid"
            backgroundColor="#3e4771"
            id={"preview-tooltip"}
            data-html
          >
            <span className={`software-name tooltip__tooltip-text`}>
              <p>
                The &quot;snapshot&quot; key includes the query&apos;s
                <br />
                results. These will be unique to your query.
              </p>
            </span>
          </ReactTooltip>
        </p>
        <div className={`${baseClass}__host-status-webhook-preview`}>
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

export default PreviewDataModal;
