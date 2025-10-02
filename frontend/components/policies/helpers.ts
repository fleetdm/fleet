import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";
import { PolicyStatus } from "pages/hosts/details/cards/Policies/HostPoliciesTable/HostPoliciesTableConfig";

const POLICY_STATUS_TO_INDICATOR_PARAMS: Record<
  PolicyStatus,
  [IndicatorStatus, string]
> = {
  pass: ["success", "Pass"],
  fail: ["failure", "Fail"],
  actionRequired: ["actionRequired", "Action required"],
};

export default POLICY_STATUS_TO_INDICATOR_PARAMS;
