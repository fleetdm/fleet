import { MdmProfileStatus } from "interfaces/mdm";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";

interface IAggregateDisplayOption {
  value: MdmProfileStatus;
  text: string;
  iconName: IndicatorStatus;
  tooltipText: string;
}

const AGGREGATE_STATUS_DISPLAY_OPTIONS: IAggregateDisplayOption[] = [
  {
    value: "verified",
    text: "Verified",
    iconName: "success",
    tooltipText: "These hosts applied all OS settings. Fleet verified.",
  },
  {
    value: "verifying",
    text: "Verifying",
    iconName: "successPartial",
    tooltipText:
      "These hosts acknowledged all MDM commands to apply OS settings. Fleet is verifying the OS settings are applied.",
  },
  {
    value: "pending",
    text: "Pending",
    iconName: "pendingPartial",
    tooltipText:
      "These hosts will apply the latest OS settings. Click on a host to view which settings.",
  },
  {
    value: "failed",
    text: "Failed",
    iconName: "error",
    tooltipText:
      "These host failed to apply the latest OS settings. Click on a host to view error(s).",
  },
];

export default AGGREGATE_STATUS_DISPLAY_OPTIONS;
