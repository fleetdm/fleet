import PropTypes from "prop-types";

export default PropTypes.shape({
  http_status: PropTypes.number,
  base: PropTypes.string,
});

export interface IError {
  name: string;
  reason: string;
}

export interface IApiError {
  message: string;
  errors: IError[];
}
