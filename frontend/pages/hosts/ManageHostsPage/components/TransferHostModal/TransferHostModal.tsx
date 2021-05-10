import React, { useCallback, useState } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";
import Modal from "components/modals/Modal";
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
}

const baseClass = "transfer-host-modal";

const TransferHostModal = (props: ITransferHostModal): JSX.Element => {
  const { onCancel, onSubmit, teams, isGlobalAdmin } = props;

  const [selectedTeam, setSelectedTeam] = useState<ITeam>();

  const onChangeSelectTeam = useCallback(
    (teamId: number) => {
      const teamWithId = teams.find((team) => team.id === teamId);
      setSelectedTeam(teamWithId as ITeam);
    },
    [teams, setSelectedTeam]
  );

  const onSubmitTransferHost = useCallback(() => {
    onSubmit(selectedTeam as ITeam);
  }, [onSubmit, selectedTeam]);

  const createTeamDropdownOptions = () => {
    return teams.map((team) => {
      return {
        value: team.id,
        label: team.name,
      };
    });
  };

  return (
    <Modal onExit={onCancel} title={"Transfer hosts"} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <Dropdown
          clearable
          wrapperClassName={`${baseClass}__team-dropdown-wrapper`}
          label={"Transfer selected hosts to:"}
          value={selectedTeam && selectedTeam.id}
          options={createTeamDropdownOptions()}
          onChange={onChangeSelectTeam}
          placeholder={"No team"}
          searchable={false}
        />
        {isGlobalAdmin ? (
          <p>
            Team not here?{" "}
            <Link to={PATHS.ADMIN_TEAMS} className={`${baseClass}__team-link`}>
              Create a team
            </Link>
          </p>
        ) : null}
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            disabled={selectedTeam === undefined}
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onSubmitTransferHost}
          >
            Transfer
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default TransferHostModal;
