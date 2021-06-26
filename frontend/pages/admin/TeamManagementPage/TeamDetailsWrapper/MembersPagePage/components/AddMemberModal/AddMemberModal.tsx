import React, { useCallback, useState } from "react";

import { INewMembersBody, ITeam } from "interfaces/team";
import endpoints from "fleet/endpoints";
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import AutocompleteDropdown from "pages/admin/TeamManagementPage/TeamDetailsWrapper/MembersPagePage/components/AutocompleteDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";

const baseClass = "add-member-modal";

interface IAddMemberModal {
  team: ITeam;
  disabledMembers: number[];
  onCancel: () => void;
  onSubmit: (userIds: INewMembersBody) => void;
}

const AddMemberModal = (props: IAddMemberModal): JSX.Element => {
  const { disabledMembers, onCancel, onSubmit, team } = props;

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
        <AutocompleteDropdown
          team={team}
          id={"member-autocomplete"}
          resourceUrl={endpoints.USERS}
          onChange={onChangeDropdown}
          placeholder={"Search users by name"}
          disabledOptions={disabledMembers}
          value={selectedMembers}
        />
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
