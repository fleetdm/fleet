import PropTypes from "prop-types";

export default PropTypes.shape({
  http_status: PropTypes.number,
  base: PropTypes.string,
});

// Response created by utilities/format_error_response
export interface IOldApiError {
  http_status: number;
  base: string;
}

export interface IError {
  name: string;
  reason: string;
}

// Response returned by API when there is an error
export interface IApiError {
  message: string;
  errors: IError[];
  uuid?: string;
}
