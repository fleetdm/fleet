import React, { useCallback, useState } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { ITeam } from "interfaces/team";

interface ITransferHostModal {
  isGlobalAdmin: boolean;
  teams: ITeam[];
  onSubmit: (team: ITeam) => void;
  onCancel: () => void;
  isUpdating: boolean;
  /** Manage host page only */
  multipleHosts?: boolean;
}

interface INoTeamOption {
  id: string;
}

const baseClass = "transfer-host-modal";

const NO_TEAM_OPTION = {
  value: "no-team",
  label: "No team",
};

const TransferHostModal = ({
  onCancel,
  onSubmit,
  teams,
  isGlobalAdmin,
  isUpdating,
  multipleHosts,
}: ITransferHostModal): JSX.Element => {
  const [selectedTeam, setSelectedTeam] = useState<ITeam | INoTeamOption>();

  const onChangeSelectTeam = useCallback(
    (teamId: number | string) => {
      if (teamId === "no-team") {
        setSelectedTeam({ id: NO_TEAM_OPTION.value });
      } else {
        const teamWithId = teams.find((team) => team.id === teamId);
        setSelectedTeam(teamWithId as ITeam);
      }
    },
    [teams, setSelectedTeam]
  );

  const onSubmitTransferHost = useCallback(() => {
    onSubmit(selectedTeam as ITeam);
  }, [onSubmit, selectedTeam]);

  const createTeamDropdownOptions = () => {
    const teamOptions = teams.map((team) => {
      return {
        value: team.id,
        label: team.name,
      };
    });
    return [NO_TEAM_OPTION, ...teamOptions];
  };

  return (
    <Modal onExit={onCancel} title="Transfer hosts" className={baseClass}>
      <form className={`${baseClass}__form`}>
        <Dropdown
          wrapperClassName={`${baseClass}__team-dropdown-wrapper`}
          label={`Transfer ${multipleHosts ? "selected hosts" : "host"} to:`}
          value={selectedTeam && selectedTeam.id}
          options={createTeamDropdownOptions()}
          onChange={onChangeSelectTeam}
          placeholder="Select a team"
          searchable={false}
          autoFocus
        />
        {isGlobalAdmin ? (
          <p>
            Team not here?{" "}
            <Link to={PATHS.ADMIN_TEAMS} className={`${baseClass}__team-link`}>
              Create a team
            </Link>
          </p>
        ) : null}
        <div className="modal-cta-wrap">
          <Button
            disabled={selectedTeam === undefined}
            type="button"
            variant="brand"
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
