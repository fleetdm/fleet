import PropTypes from "prop-types";

export default PropTypes.shape({
  http_status: PropTypes.number,
  base: PropTypes.string,
});

interface IError {
  name: string;
  reason: string;
}

export interface IApiError {
  message: string;
  errors: IError[];
}
