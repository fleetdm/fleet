import PropTypes from "prop-types";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.oneOfType([PropTypes.number]),
  name: PropTypes.string,
  query: PropTypes.string,
  label_type: PropTypes.oneOf(["regular", "builtin"]),
  label_membership_type: PropTypes.oneOf(["dynamic", "manual"]),
  hosts_count: PropTypes.number,
  display_text: PropTypes.string,
  count: PropTypes.number, // seems to be a repeat of hosts_count issue #1618
  host_ids: PropTypes.arrayOf(PropTypes.number),
});

export type LabelType = "regular" | "builtin";
export type LabelMembershipType = "dynamic" | "manual";

export interface ILabelSummary {
  id: number;
  name: string;
  description?: string;
  label_type: LabelType;
}

export interface ILabelSoftwareTitle {
  id: number;
  name: string;
}

export interface ILabel extends ILabelSummary {
  created_at: string;
  updated_at: string;
  uuid?: string;
  query: string;
  label_membership_type: LabelMembershipType;
  host_count?: number; // returned for built-in labels but not custom labels
  display_text: string;
  count: number; // seems to be a repeat of hosts_count issue #1618
  host_ids: number[] | null;
  type?: "custom" | "platform" | "status" | "all";
  slug?: string; // e.g., "labels/13" | "online"
  target_type?: string; // e.g., "labels"
  platform: string;
}

// corresponding to fleet>server>fleet>labels.go>LabelSpec
export interface ILabelSpecResponse {
  specs: {
    id: number;
    name: string;
    description: string;
    query: string;
    platform?: string; // improve to only allow possible platforms from API
    label_type?: LabelType;
    label_membership_type: LabelMembershipType;
    hosts?: string[];
  };
}
