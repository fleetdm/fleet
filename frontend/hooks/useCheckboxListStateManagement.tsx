import { useState } from "react";

import { IPolicy } from "interfaces/policy";

interface ICheckedPolicy {
  name?: string;
  id: number;
  isChecked: boolean;
}

const useCheckboxListStateManagement = (
  allPolicies: IPolicy[],
  automatedPolicies: number[] | undefined
) => {
  const [policyItems, setPolicyItems] = useState<ICheckedPolicy[]>(() => {
    return allPolicies.map(({ name, id }) => ({
      name,
      id,
      isChecked: !!automatedPolicies?.includes(id),
    }));
  });

  const updatePolicyItems = (policyId: number) => {
    setPolicyItems((prevItems) =>
      prevItems.map((policy) =>
        policy.id !== policyId
          ? policy
          : { ...policy, isChecked: !policy.isChecked }
      )
    );
  };

  return { policyItems, updatePolicyItems };
};

export default useCheckboxListStateManagement;
