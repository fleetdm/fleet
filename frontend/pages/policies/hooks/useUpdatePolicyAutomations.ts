import { useMutation } from "react-query";

import { IPolicy, IPolicyFormData } from "interfaces/policy";
import policiesAPI from "services/entities/policies";

/** The automation fields of a policy update. */
export type IPolicyAutomationUpdate = Pick<
  IPolicyFormData,
  | "automation_type"
  | "software_title_id"
  | "software_package_id"
  | "script_id"
  | "profile_uuid"
  | "continuous_automations_enabled"
  | "force_patch"
  | "patch_when_closed"
  | "notify_before_patching"
  | "patch_deadline"
>;

export interface IUpdatePolicyAutomationsVars {
  policyUpdate: IPolicyAutomationUpdate;
}

interface IUseUpdatePolicyAutomationsArgs {
  /** May be `undefined` during initial render / for a not-yet-loaded policy;
   *  the mutation guards against this and is only invokable once it's set. */
  policy: IPolicy | undefined;
  onSuccess?: () => void;
  onError?: (err: unknown) => void;
}

/** Saves a policy's automation in one request to the policy. */
const useUpdatePolicyAutomations = ({
  policy,
  onSuccess,
  onError,
}: IUseUpdatePolicyAutomationsArgs) =>
  useMutation(
    ({ policyUpdate }: IUpdatePolicyAutomationsVars) => {
      if (!policy) {
        return Promise.reject(
          new Error("Cannot update automations without a policy.")
        );
      }
      return policiesAPI.update(policy.id, policyUpdate);
    },
    { onSuccess, onError }
  );

export default useUpdatePolicyAutomations;
