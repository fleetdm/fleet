import React, { useState } from "react";
import { Dispatch } from "redux";

import {
  setSelectedTargets,
  setSelectedTargetsQuery, // @ts-ignore
} from "redux/nodes/components/QueryPages/actions";
import { ITarget, ITargetsResponse } from "interfaces/target";
import { ICampaign } from "interfaces/campaign";

 // @ts-ignore
import SelectTargetsDropdown from "components/forms/fields/SelectTargetsDropdown";

interface ISelectTargetsProps {
  selectedTargets: ITarget[];
  campaign: ICampaign | null;
  isBasicTier: boolean;
  queryIdForEdit: string | undefined;
  dispatch: Dispatch;
};

const baseClass = "query-page-select-targets";

const SelectTargets = ({
  selectedTargets,
  campaign,
  isBasicTier,
  queryIdForEdit,
  dispatch,
}: ISelectTargetsProps) => {
  const [targetsCount, setTargetsCount] = useState<number>(0);
  const [targetsError, setTargetsError] = useState<string | null>(null);

  const onFetchTargets = (
    targetSearchText: string,
    targetResponse: ITargetsResponse
  ) => {
    const { targets_count: responseTargetsCount } = targetResponse;

    dispatch(setSelectedTargetsQuery(targetSearchText));
    setTargetsCount(responseTargetsCount);

    return false;
  };

  const onTargetSelect = (selected: ITarget[]) => {
    setTargetsError(null);
    dispatch(setSelectedTargets(selectedTargets));

    return false;
  };

  // TODO: Figure out where to put this
  // if (!targetsCount) {
  //   setTargetsError(
  //     "You must select a target with at least one host to run a query"
  //   );

  //   return false;
  // }

  return (
    <div className={`${baseClass}__wrapper body-wrap`}>
      <SelectTargetsDropdown
        error={targetsError}
        onFetchTargets={onFetchTargets}
        onSelect={onTargetSelect}
        selectedTargets={selectedTargets}
        targetsCount={targetsCount}
        label="Select targets"
        queryId={queryIdForEdit}
        isBasicTier={isBasicTier}
      />
    </div>
  );
};

export default SelectTargets;