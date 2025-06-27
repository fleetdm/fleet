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
export type LabelMembershipType = "dynamic" | "manual" | "host_vitals";
export type LabelHostVitalsCriterion =
  | "end_user_idp_group"
  | "end_user_idp_department"; // for now, may expand to be configurable

export type LabelLeafCriterion = {
  vital: LabelHostVitalsCriterion;
  value: string; // from user input
};

type LabelAndCriterion = {
  and: LabelHostVitalsCriteria[];
};

type LabelOrCriterion = {
  or: LabelHostVitalsCriteria[];
};

export type LabelHostVitalsCriteria =
  | LabelLeafCriterion
  | LabelAndCriterion
  | LabelOrCriterion;

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

export interface ILabelQuery {
  id: number;
  name: string;
}

export interface ILabelPolicy {
  id: number;
  name: string;
}

export interface ILabel extends ILabelSummary {
  created_at: string;
  updated_at: string;
  uuid?: string;
  host_count?: number; // returned for built-in labels but not custom labels
  display_text: string;
  count: number; // seems to be a repeat of hosts_count issue #1618
  type?: "custom" | "platform" | "status" | "all";
  slug?: string; // e.g., "labels/13" | "online"
  target_type?: string; // e.g., "labels"
  author_id?: number;

  label_membership_type: LabelMembershipType;
  // dynamic-specific
  query: string; // does return '""' for other types
  platform: string; // does return '""' for other types
  // host_vitals-specific
  criteria: LabelHostVitalsCriteria | null;

  // manual-specific
  host_ids: number[] | null;
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
