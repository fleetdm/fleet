import React, { useState } from "react";

import { humanHostDetailUpdated } from "fleet/helpers";

// @ts-ignore
import { stringToClipboard } from "utilities/copy_text";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";

import { ITeam } from "interfaces/team";

import CopyIcon from "../../../../../../../assets/images/icon-copy-clipboard-fleet-blue-20x20@2x.png";

interface IRenderOSPolicyModal {
  onCreateNewPolicy: (team: ITeam) => void;
  onCancel: () => void;
  titleData?: any;
  osPolicy: string;
  osPolicyLabel: string;
}

const baseClass = "render-os-policy-modal";

const RenderOSPolicyModal = ({
  onCancel,
  onCreateNewPolicy,
  titleData,
  osPolicy,
  osPolicyLabel,
}: IRenderOSPolicyModal): JSX.Element => {
  const [copyMessage, setCopyMessage] = useState<string>("");

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
            {copyMessage && <span>{`${copyMessage} `}</span>}
            <Button
              variant="unstyled"
              className={`${baseClass}__os-policy-copy-icon`}
              onClick={onCopyOsPolicy}
            >
              <img src={CopyIcon} alt="copy" />
            </Button>
          </span>
        </span>
      </div>
    );
  };

  return (
    <Modal
      title="Operating system"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <>
        <p>
          <span className={`${baseClass}__os-modal-title`}>
            {titleData.os_version}{" "}
          </span>
          <span className={`${baseClass}__os-modal-updated`}>
            Reported {humanHostDetailUpdated(titleData.detail_updated_at)}
          </span>
        </p>
        <span className={`${baseClass}__os-modal-example-title`}>
          <TooltipWrapper tipContent="A policy is a yes or no question you can ask all your devices.">
            Example policy:
          </TooltipWrapper>
        </span>
        <InputField
          disabled
          inputWrapperClass={`${baseClass}__os-policy`}
          name="os-policy"
          label={renderOsPolicyLabel()}
          type={"textarea"}
          value={osPolicy}
        />
        <div className={"modal-btn-wrap"}>
          <Button onClick={onCancel} variant="inverse">
            Close
          </Button>
          <Button onClick={onCreateNewPolicy} variant="brand">
            Create new policy
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default RenderOSPolicyModal;
