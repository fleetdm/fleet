import React, { useCallback, useContext, useState } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import useGitOpsMode from "hooks/useGitOpsMode";
import mdmAppleAPI from "services/entities/mdm_apple";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import VppFleetsField from "../VppFleetsField";
import {
  DEFAULT_VPP_FLEETS_VALUE,
  IVppFleetsValue,
  teamIdsForApi,
} from "../VppFleetsField/helpers";

import { getErrorMessage } from "./helpers";

const baseClass = "add-vpp-modal";

interface IAddVppModalProps {
  onCancel: () => void;
  onAdded: () => void;
}

const AddVppModal = ({ onCancel, onAdded }: IAddVppModalProps) => {
  const { availableTeams } = useContext(AppContext);
  // Fleet assignments are managed in GitOps, so tokens uploaded in GitOps mode stay unassigned.
  const { gitOpsModeEnabled } = useGitOpsMode();

  const [tokenFile, setTokenFile] = useState<File | null>(null);
  const [fleets, setFleets] = useState<IVppFleetsValue>(
    DEFAULT_VPP_FLEETS_VALUE
  );
  const [isUploading, setIsUploading] = useState(false);

  const onSelectFile = useCallback((files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  }, []);

  const uploadVppToken = useCallback(async () => {
    setIsUploading(true);
    if (!tokenFile) {
      setIsUploading(false);
      notify.error("No token selected.");
      return;
    }

    let tokenId: number;
    try {
      const { token } = await mdmAppleAPI.uploadVppToken(tokenFile);
      tokenId = token.id;
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
      setIsUploading(false);
      onCancel();
      return;
    }

    if (!gitOpsModeEnabled) {
      try {
        await mdmAppleAPI.editVppTeams({
          tokenId,
          teamIds: teamIdsForApi(fleets),
        });
      } catch (e) {
        notify.error(
          "VPP token added, but couldn't assign fleets. Please edit fleets and try again.",
          { response: e }
        );
        setIsUploading(false);
        onAdded();
        return;
      }
    }

    notify.success("Added successfully.");
    setIsUploading(false);
    onAdded();
  }, [tokenFile, gitOpsModeEnabled, fleets, onAdded, onCancel]);

  return (
    <Modal
      className={baseClass}
      title="Add VPP"
      onExit={onCancel}
      width="large"
    >
      <>
        <p className={`${baseClass}__description`}>
          Follow the step-by-step guide to add VPP.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/add-vpp`}
            text="Learn more"
            newTab
          />
        </p>
        <FileUploader
          className={`${baseClass}__file-uploader ${
            isUploading ? `${baseClass}__file-uploader--loading` : ""
          }`}
          accept=".vpptoken"
          message="Content token (.vpptoken)"
          graphicName="file-vpp"
          buttonType="secondary"
          buttonMessage={isUploading ? "Uploading..." : "Upload"}
          fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
          onFileUpload={onSelectFile}
        />
        {!gitOpsModeEnabled && (
          <VppFleetsField
            value={fleets}
            onChange={setFleets}
            availableTeams={availableTeams || []}
            disabled={isUploading}
            tooltipPosition="bottom"
            tooltip={
              <>
                Choose <b>Custom</b> if you want your App Store apps from this
                VPP token to only be available to install on hosts in specific
                fleets.{" "}
                {/* TODO: point at the page product picks for this tooltip */}
                <CustomLink
                  url={`${LEARN_MORE_ABOUT_BASE_LINK}/vpp-fleets`}
                  text="Learn more"
                  newTab
                  variant="tooltip-link"
                />
              </>
            }
          />
        )}
        <div className="modal-cta-wrap">
          <Button
            onClick={uploadVppToken}
            isLoading={isUploading}
            disabled={!tokenFile || isUploading}
          >
            Add VPP
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddVppModal;
