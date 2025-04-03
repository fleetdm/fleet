import React, { useState } from "react";

import { stringToClipboard } from "utilities/copy_text";
import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "fleet-app-details-modal";

interface IFleetAppDetailsModalProps {
  name: string;
  platform: string;
  version: string;
  slug?: string;
  url?: string;
  onCancel: () => void;
}

const TOOLTIP_MESSAGE =
  "Fleet downloads the package from the URL and stores it. Hosts download it from Fleet before install.";

const FleetAppDetailsModal = ({
  name,
  platform,
  version,
  slug,
  url,
  onCancel,
}: IFleetAppDetailsModalProps) => {
  const [copyMessage, setCopyMessage] = useState("");

  const onCopySlug = (evt: React.MouseEvent) => {
    evt.preventDefault();

    stringToClipboard(slug)
      .then(() => setCopyMessage("Copied!"))
      .catch(() => setCopyMessage("Copy failed"));

    // Clear message after 1 second
    setTimeout(() => setCopyMessage(""), 1000);

    return false;
  };

  return (
    <Modal className={baseClass} title="Software details" onExit={onCancel}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <DataSet title="Name" value={name} />
          <DataSet title="Version" value={version} />
          {slug && (
            <DataSet
              title="Fleet-maintained app slug"
              value={
                <>
                  {slug}{" "}
                  <div className={`${baseClass}__action-overlay`}>
                    {copyMessage && (
                      <div
                        className={`${baseClass}__copy-message`}
                      >{`${copyMessage} `}</div>
                    )}
                  </div>
                  <Button
                    variant="unstyled"
                    className={`${baseClass}__copy-secret-icon`}
                    onClick={onCopySlug}
                  >
                    <Icon name="copy" />
                  </Button>
                </>
              }
            />
          )}
          <DataSet title="Platform" value={PLATFORM_DISPLAY_NAMES[platform]} />

          {url && (
            <DataSet
              title={
                <TooltipWrapper tipContent={TOOLTIP_MESSAGE}>
                  URL
                </TooltipWrapper>
              }
              value={<TooltipTruncatedText value={url} />}
            />
          )}
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

export default FleetAppDetailsModal;
