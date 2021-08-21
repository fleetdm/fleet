/* This component is used for creating policies */

import React, { useState, useCallback } from "react";
// @ts-ignore
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { IQuery } from "interfaces/query";

const baseClass = "add-policy-modal";

interface IAddPolicyModalProps {
  allQueries: IQuery[];
  onCancel: () => void;
  onSubmit: (query_id: any) => void;
}

const AddPolicyModal = ({
  onCancel,
  onSubmit,
  allQueries,
}: IAddPolicyModalProps): JSX.Element => {
  const [selectedQuery, setSelectedQuery] = useState<number | undefined>();

  const createQueryDropdownOptions = () => {
    const queryOptions = allQueries.map((q) => {
      return {
        value: q.id, // TODO confirm if string or number?
        label: q.name,
      };
    });
    return queryOptions;
  };

  const onChangeSelectQuery = useCallback(
    (queryId: number) => {
      setSelectedQuery(queryId);
    },
    [setSelectedQuery]
  );

  return (
    <Modal title={"Add a policy"} onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <Dropdown
          searchable
          options={createQueryDropdownOptions()}
          onChange={onChangeSelectQuery}
          placeholder={"Select query"}
          value={selectedQuery}
          wrapperClassName={`${baseClass}__select-query-dropdown-wrapper`}
        />
        <InfoBanner className={`${baseClass}__sandbox-info`}>
          <p>
            Host that return results for the selected query are <b>Passing</b>.
          </p>

          <p>
            Hosts that do not return results for the selected query are{" "}
            <b>Failing</b>.
          </p>

          <p>
            To test which hosts return results, it is recommened to first run
            your query as a live query by heading to <b>Queries</b> and then
            selecting a query.
          </p>
        </InfoBanner>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onSubmit}
            disabled={!selectedQuery}
          >
            Add
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

export default AddPolicyModal;
