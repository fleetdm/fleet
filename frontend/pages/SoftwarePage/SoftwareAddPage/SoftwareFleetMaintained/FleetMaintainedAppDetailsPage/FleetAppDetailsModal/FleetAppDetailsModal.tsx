import React from "react";

import {
  LEARN_MORE_ABOUT_BASE_LINK,
  PLATFORM_DISPLAY_NAMES,
} from "utilities/constants";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
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
                {slug} <CopyButton copyText={slug} variant="compact" />
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
        <Button onClick={onCancel}>Close</Button>
      </div>
    </Modal>
  );
};

export default FleetAppDetailsModal;
