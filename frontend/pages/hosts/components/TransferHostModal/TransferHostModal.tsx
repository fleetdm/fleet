import React, { useCallback, useState } from "react";

import PATHS from "router/paths";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import CustomLink from "components/CustomLink";
import { ITeam } from "interfaces/team";

interface ITransferHostModal {
  isGlobalAdmin: boolean;
  teams: ITeam[];
  onSubmit: (team: ITeam) => void;
  onCancel: () => void;
  isUpdating: boolean;
  multipleHosts?: boolean;
  hostsTeamId?: number | null;
}

interface INoTeamOption {
  id: string;
}

const baseClass = "transfer-host-modal";

const NO_TEAM_OPTION = {
  value: "no-team",
  label: "Unassigned",
};

const TransferHostModal = ({
  onCancel,
  onSubmit,
  teams,
  isGlobalAdmin,
  isUpdating,
  multipleHosts,
  hostsTeamId,
}: ITransferHostModal): JSX.Element => {
  const [selectedTeam, setSelectedTeam] = useState<ITeam | INoTeamOption>();

  const getDropdownValue = () => {
    if (!selectedTeam) {
      return undefined;
    }

    if ("id" in selectedTeam && selectedTeam.id === "no-team") {
      return "no-team";
    }

    return String((selectedTeam as ITeam).id);
  };

  const onChangeSelectTeam = useCallback(
    (newValue: CustomOptionType | null) => {
      if (!newValue) {
        setSelectedTeam(undefined);
        return;
      }

      if (newValue.value === "no-team") {
        setSelectedTeam({ id: NO_TEAM_OPTION.value });
        return;
      }

      // newValue.value is a string; team.id is number, so coerce
      const teamId = Number(newValue.value);
      const teamWithId = teams.find((team) => team.id === teamId);
      setSelectedTeam(teamWithId as ITeam);
    },
    [teams]
  );

  const onSubmitTransferHost = useCallback(() => {
    onSubmit(selectedTeam as ITeam);
  }, [onSubmit, selectedTeam]);

  const createTeamDropdownOptions = (): CustomOptionType[] => {
    const teamOptions: CustomOptionType[] = teams
      .filter((team) => team.id !== hostsTeamId)
      .map((team) => ({
        value: String(team.id),
        label: team.name,
      }));

    // Hosts on no team cannot transfer to no team again
    const canTransferToNoTeam = hostsTeamId !== 0 && hostsTeamId !== null;

    return canTransferToNoTeam
      ? [NO_TEAM_OPTION as CustomOptionType, ...teamOptions]
      : teamOptions;
  };

  return (
    <Modal onExit={onCancel} title="Transfer" className={baseClass}>
      <form className={`${baseClass}__form`}>
        <DropdownWrapper
          name="transfer-team"
          wrapperClassname={`${baseClass}__team-dropdown-wrapper`}
          label={`Transfer ${multipleHosts ? "selected hosts" : "host"} to:`}
          value={getDropdownValue()}
          options={createTeamDropdownOptions()}
          onChange={onChangeSelectTeam}
          placeholder="Select a fleet"
          isSearchable
        />
        {isGlobalAdmin ? (
          <p>
            Fleet not here?{" "}
            <CustomLink
              url={PATHS.ADMIN_FLEETS}
              className={`${baseClass}__team-link`}
              text="Create a fleet"
            />
          </p>
        ) : null}
        <div className="modal-cta-wrap">
          <Button
            disabled={selectedTeam === undefined}
            type="button"
            onClick={onSubmitTransferHost}
            className="transfer-loading"
            isLoading={isUpdating}
          >
            Transfer
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default TransferHostModal;
