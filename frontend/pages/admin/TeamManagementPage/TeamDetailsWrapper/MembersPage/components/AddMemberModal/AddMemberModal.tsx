import React, { useCallback, useState } from "react";

import { INewMembersBody, ITeam } from "interfaces/team";
import { IUser } from "interfaces/user";
import endpoints from "fleet/endpoints";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import AutocompleteDropdown from "pages/admin/TeamManagementPage/TeamDetailsWrapper/MembersPage/components/AutocompleteDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";

const baseClass = "add-member-modal";

interface IAddMemberModal {
  team: ITeam;
  disabledMembers: number[];
  onCancel: () => void;
  onSubmit: (userIds: INewMembersBody) => void;
  onCreateNewMember: () => void;
}

const AddMemberModal = ({
  disabledMembers,
  onCancel,
  onSubmit,
  onCreateNewMember,
  team,
}: IAddMemberModal): JSX.Element => {
  const [selectedMembers, setSelectedMembers] = useState([]);

  const onChangeDropdown = useCallback(
    (values) => {
      setSelectedMembers(values);
    },
    [setSelectedMembers]
  );

  const onFormSubmit = useCallback(() => {
    const newMembers = selectedMembers.map((member: IDropdownOption) => {
      return { id: member.value as number, role: "observer" };
    });
    onSubmit({ users: newMembers });
  }, [selectedMembers, onSubmit]);

  return (
    <Modal onExit={onCancel} title={"Add Members"} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <p className="title">Add team members</p>
        <AutocompleteDropdown
          team={team}
          id={"member-autocomplete"}
          resourceUrl={endpoints.USERS}
          onChange={onChangeDropdown}
          placeholder={"Search users by name"}
          disabledOptions={disabledMembers}
          value={selectedMembers}
        />
        <p>
          User not here?&nbsp;
          <Button
            onClick={onCreateNewMember}
            variant={"text-link"}
            className={"light-text"}
          >
            <>
              <strong>Create a user</strong>
            </>
          </Button>
        </p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            disabled={selectedMembers.length === 0}
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
          >
            Add Member
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

export default AddMemberModal;
