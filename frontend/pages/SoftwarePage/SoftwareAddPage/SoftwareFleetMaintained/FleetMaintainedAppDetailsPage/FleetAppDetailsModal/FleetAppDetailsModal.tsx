import React, { useState } from "react";

import { stringToClipboard } from "utilities/copy_text";
import {
  LEARN_MORE_ABOUT_BASE_LINK,
  PLATFORM_DISPLAY_NAMES,
} from "utilities/constants";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import CustomLink from "components/CustomLink";

const baseClass = "fleet-app-details-modal";

interface IFleetAppDetailsModalProps {
  name: string;
  platform: string;
  version: string;
  slug?: string;
  url?: string;
  onCancel: () => void;
}

const SLUG_TOOLTIP_MESSAGE = (
  <>
    Used to manage apps in Gitops.{" "}
    <CustomLink
      newTab
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/gitops`}
      text="Learn more"
      variant="tooltip-link"
    />
  </>
);

const URL_TOOLTIP_MESSAGE = (
  <>
    Fleet downloads the package from the URL and stores it.
    <br />
    Hosts download it from Fleet before install.
  </>
);

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

  let versionElement = <>{version}</>;
  if (version === "latest") {
    versionElement = (
      <TooltipWrapper
        tipContent={
          <>
            To preview the version download
            <br />
            {name} using the URL below.
          </>
        }
      >
        Latest
      </TooltipWrapper>
    );
  }

  return (
    <Modal className={baseClass} title="Software details" onExit={onCancel}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <DataSet title="Name" value={name} />
          <DataSet title="Version" value={versionElement} />
          {slug && (
            <DataSet
              title={
                <TooltipWrapper tipContent={SLUG_TOOLTIP_MESSAGE}>
                  Fleet-maintained app slug
                </TooltipWrapper>
              }
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
                <TooltipWrapper tipContent={URL_TOOLTIP_MESSAGE}>
                  URL
                </TooltipWrapper>
              }
              value={<TooltipTruncatedText value={url} />}
            />
          )}
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default FleetAppDetailsModal;
