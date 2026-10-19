import React, { useCallback, useContext, useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { IMdmVppToken } from "interfaces/mdm";
import mdmAppleAPI from "services/entities/mdm_apple";

import VppFleetsField from "../VppFleetsField";
import {
  IVppFleetsValue,
  teamIdsForApi,
  vppFleetsValueFromToken,
} from "../VppFleetsField/helpers";

const baseClass = "edit-teams-vpp-modal";

interface IEditTeamsVppModalProps {
  currentToken: IMdmVppToken;
  onCancel: () => void;
  onSuccess: () => void;
}

const EditTeamsVppModal = ({
  currentToken,
  onCancel,
  onSuccess,
}: IEditTeamsVppModalProps) => {
  const { availableTeams } = useContext(AppContext);

  const [fleets, setFleets] = useState<IVppFleetsValue>(
    vppFleetsValueFromToken(currentToken)
  );
  const [isSaving, setIsSaving] = useState(false);

  const onSave = useCallback(
    async (evt: React.FormEvent<HTMLFormElement>) => {
      evt.preventDefault();
      setIsSaving(true);
      try {
        await mdmAppleAPI.editVppTeams({
          tokenId: currentToken.id,
          teamIds: teamIdsForApi(fleets),
        });
        notify.success("Edited successfully.");
        onSuccess();
      } catch (e) {
        notify.error("Couldn’t edit. Please try again.", { response: e });
      } finally {
        setIsSaving(false);
      }
    },
    [currentToken.id, fleets, onSuccess]
  );

  return (
    <Modal
      className={baseClass}
      title="Edit fleets"
      onExit={onCancel}
      width="large"
      isContentDisabled={isSaving}
    >
      <form onSubmit={onSave} className={baseClass} autoComplete="off">
        <p className={`${baseClass}__description`}>
          Edit fleets for <b>{currentToken.org_name}</b>.
        </p>
        <VppFleetsField
          value={fleets}
          onChange={setFleets}
          availableTeams={availableTeams || []}
          tooltipPosition="top"
          tooltip={
            <>
              Choose <b>Custom</b> if you want your App Store apps from this VPP
              token to only be available to install on hosts in specific fleets.
            </>
          }
          helpText="If you uncheck a fleet, apps from this VPP token will be deleted from that fleet. Installed apps won't be uninstalled from hosts."
        />
        <div className="modal-cta-wrap">
          <Button type="submit" isLoading={isSaving}>
            Save
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditTeamsVppModal;
