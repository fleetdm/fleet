import PropTypes, { string } from "prop-types";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
  name: PropTypes.string,
  query: PropTypes.string,
  label_type: PropTypes.string,
  label_membership_type: PropTypes.string,
  hosts_count: PropTypes.number,
  display_text: PropTypes.string,
  count: PropTypes.number, // seems to be a repeat of hosts_count issue #1618
  host_ids: PropTypes.arrayOf(PropTypes.number),
});

export interface ILabel {
  created_at: string;
  updated_at: string;
  id: number | string;
  name: string;
  description?: string;
  query: string;
  label_type: "regular" | "builtin";
  label_membership_type: string;
  hosts_count: number;
  display_text: string;
  count: number; // seems to be a repeat of hosts_count issue #1618
  host_ids: number[] | null;
  type: "custom" | "platform" | "status" | "all";
  slug?: string; // e.g., "labels/13" | "online"
  target_type?: string; // e.g., "labels"
  platform: string;
}

export interface ILabelFormData {
  name: string;
  query: string;
  description: string;
  platform: string;
}
