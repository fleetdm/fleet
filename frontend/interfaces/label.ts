import PropTypes, { string } from "prop-types";

export default PropTypes.shape({
  created_at: PropTypes.string,
  description: PropTypes.string,
  display_text: PropTypes.string,
  hosts_count: PropTypes.number,
  host_ids: PropTypes.arrayOf(
    PropTypes.oneOfType([PropTypes.number, PropTypes.string])
  ),
  id: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
  name: PropTypes.string,
  label_type: PropTypes.string,
  title: PropTypes.string, // confirm on rest api doc
  type: PropTypes.string, // confirm on rest api doc
  count: PropTypes.number, // confirm on rest api doc
});

export interface ILabel {
  created_at: string;
  description: string;
  display_text: string;
  hosts_count: number;
  host_ids: number[] | null;
  id: number | string;
  label_membership_type: string;
  label_type: string;
  name: string;
  query: string;
  updated_at: string;
  title: string; // confirm on rest api doc
  type: string; // confirm on rest api doc
  count: number;
}
