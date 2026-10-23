import React, { useCallback, useContext, useState } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import mdmAppleAPI from "services/entities/mdm_apple";

import VppFleetsSelector from "../VppFleetsSelector";
import { fleetsForApi, IVppFleetsValue } from "../VppFleetsSelector/helpers";

import { getErrorMessage } from "./helpers";

const baseClass = "add-vpp-modal";

interface IAddVppModalProps {
  onCancel: () => void;
  onAdded: () => void;
}

const LEARN_MORE_URL = "https://fleetdm.com/learn-more-about/add-vpp";
const FLEETS_LEARN_MORE_URL =
  "https://fleetdm.com/learn-more-about/vpp-tokens-best-practice";

const AddVppModal = ({ onCancel, onAdded }: IAddVppModalProps) => {
  const { config } = useContext(AppContext);
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

  const [tokenFile, setTokenFile] = useState<File | null>(null);
  const [fleetsValue, setFleetsValue] = useState<IVppFleetsValue>({
    allFleets: true,
    fleetIds: [],
  });
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

    try {
      const { token } = await mdmAppleAPI.uploadVppToken(tokenFile);
      const fleets = fleetsForApi(fleetsValue);
      // The new token is unassigned, so Custom with no fleets needs no PATCH.
      if (fleets !== null && !gitOpsModeEnabled) {
        try {
          await mdmAppleAPI.editVppTeams({
            tokenId: token.id,
            teamIds: fleets,
          });
        } catch (e) {
          notify.error(
            "Couldn't assign fleets. The VPP token was added. Try again from Actions > Edit fleets.",
            { response: e }
          );
          onAdded();
          return;
        }
      }
      notify.success("Added successfully.");
      onAdded();
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
      onCancel();
    } finally {
      setIsUploading(false);
    }
  }, [tokenFile, fleetsValue, gitOpsModeEnabled, onAdded, onCancel]);

  return (
    <Modal
      className={baseClass}
      title="Add VPP"
      onExit={onCancel}
      width="large"
    >
      <p className={`${baseClass}__description`}>
        Follow the step-by-step guide to add VPP.{" "}
        <CustomLink url={LEARN_MORE_URL} text="Learn more" newTab />
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
        <VppFleetsSelector
          value={fleetsValue}
          onChange={setFleetsValue}
          tooltipPosition="bottom-start"
          tooltipLearnMoreUrl={FLEETS_LEARN_MORE_URL}
          disabled={isUploading}
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
    </Modal>
  );
};

export default AddVppModal;
