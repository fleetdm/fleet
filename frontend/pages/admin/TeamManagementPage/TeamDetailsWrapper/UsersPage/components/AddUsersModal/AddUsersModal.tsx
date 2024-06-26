import React, { useCallback, useState } from "react";

import { INewTeamUser, INewTeamUsersBody, ITeam } from "interfaces/team";
import endpoints from "utilities/endpoints";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import AutocompleteDropdown from "pages/admin/TeamManagementPage/TeamDetailsWrapper/UsersPage/components/AutocompleteDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";

const baseClass = "add-user-modal";

interface IAddUsersModal {
  team: ITeam;
  disabledUsers: number[];
  onCancel: () => void;
  onSubmit: (userIds: INewTeamUsersBody) => void;
  onCreateNewTeamUser: () => void;
}

const AddUsersModal = ({
  disabledUsers,
  onCancel,
  onSubmit,
  onCreateNewTeamUser,
  team,
}: IAddUsersModal): JSX.Element => {
  const [selectedUsers, setSelectedUsers] = useState([]);

  const onChangeDropdown = useCallback(
    (values: any) => {
      setSelectedUsers(values);
    },
    [setSelectedUsers]
  );

  const onFormSubmit = useCallback(() => {
    const newUsers: INewTeamUser[] = selectedUsers.map(
      (user: IDropdownOption) => {
        return { id: user.value as number, role: "observer" };
      }
    );
    onSubmit({ users: newUsers });
  }, [selectedUsers, onSubmit]);

  return (
    <Modal onExit={onCancel} title="Add users" className={baseClass}>
      <form className={`${baseClass}__form`}>
        <div className="form-field">
          <label className="form-field__label" htmlFor="user-autocomplete">
            Grant users access to this team
          </label>
          <AutocompleteDropdown
            team={team}
            id="user-autocomplete"
            resourceUrl={endpoints.USERS}
            onChange={onChangeDropdown}
            placeholder="Search users by name"
            disabledOptions={disabledUsers}
            value={selectedUsers}
            autoFocus
          />
        </div>
        <p>
          User not here?&nbsp;
          <Button
            onClick={onCreateNewTeamUser}
            variant="text-link"
            className="light-text"
          >
            <>
              <strong>Create a user</strong>
            </>
          </Button>
        </p>
        <div className="modal-cta-wrap">
          <Button
            disabled={selectedUsers.length === 0}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
          >
            Add users
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default AddUsersModal;
