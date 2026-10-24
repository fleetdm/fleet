import PropTypes from "prop-types";

export default PropTypes.shape({
  secret: PropTypes.string,
  created_at: PropTypes.string,
  fleet_id: PropTypes.number,
});

export interface IEnrollSecret {
  secret: string;
  created_at?: string;
  fleet_id?: number;
}

export interface IEnrollSecretsResponse {
  secrets: IEnrollSecret[];
}
