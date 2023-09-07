import React, { useState } from "react";

import { humanHostDetailUpdated } from "utilities/helpers";
import { stringToClipboard } from "utilities/copy_text";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import NewTooltipWrapper from "components/NewTooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

import { ITeam } from "interfaces/team";

interface IOSPolicyModal {
  onCreateNewPolicy: (team: ITeam) => void;
  onCancel: () => void;
  osVersion?: string;
  detailsUpdatedAt?: string;
  osPolicy: string;
  osPolicyLabel: string;
}

const baseClass = "os-policy-modal";

const OSPolicyModal = ({
  onCancel,
  onCreateNewPolicy,
  osVersion,
  detailsUpdatedAt,
  osPolicy,
  osPolicyLabel,
}: IOSPolicyModal): JSX.Element => {
  const [copyMessage, setCopyMessage] = useState("");

  const renderOsPolicyLabel = () => {
    const onCopyOsPolicy = (evt: React.MouseEvent) => {
      evt.preventDefault();

      stringToClipboard(osPolicy)
        .then(() => setCopyMessage("Copied!"))
        .catch(() => setCopyMessage("Copy failed"));

      // Clear message after 1 second
      setTimeout(() => setCopyMessage(""), 1000);

      return false;
    };

    return (
      <div>
        <span className={`${baseClass}__cta`}>{osPolicyLabel}</span>{" "}
        <span className={`${baseClass}__name`}>
          <span className="buttons">
            {copyMessage && (
              <span
                className={`${baseClass}__copy-message`}
              >{`${copyMessage} `}</span>
            )}
            <Button
              variant="unstyled"
              className={`${baseClass}__os-policy-copy-icon`}
              onClick={onCopyOsPolicy}
            >
              <Icon name="copy" />
            </Button>
          </span>
        </span>
      </div>
    );
  };

  return (
    <Modal title="Operating system" onExit={onCancel} className={baseClass}>
      <>
        <p>
          <span className={`${baseClass}__os-modal-title`}>{osVersion} </span>
          <span className={`${baseClass}__os-modal-updated`}>
            Reported {humanHostDetailUpdated(detailsUpdatedAt)}
          </span>
        </p>
        <span className={`${baseClass}__os-modal-example-title`}>
          <NewTooltipWrapper tipContent="A policy is a yes or no question you can ask all your devices.">
            Example policy:
          </NewTooltipWrapper>
        </span>
        <InputField
          disabled
          inputWrapperClass={`${baseClass}__os-policy`}
          name="os-policy"
          label={renderOsPolicyLabel()}
          type={"textarea"}
          value={osPolicy}
        />
        <div className="modal-cta-wrap">
          <Button onClick={onCreateNewPolicy} variant="brand">
            Create new policy
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Close
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default OSPolicyModal;
