import PropTypes from "prop-types";

export default PropTypes.shape({
  error: PropTypes.string,
  name: PropTypes.string,
  onChange: PropTypes.func,
  value: PropTypes.oneOfType([
    PropTypes.array,
    PropTypes.bool,
    PropTypes.number,
    PropTypes.string,
  ]),
});

export interface IFormField<T = any[] | boolean | number | string> {
  error: string;
  name: string;
  onChange: (value: any) => void;
  value: T;
}

/**
 * InputField onChange receives either the field value or an object { name, value },
 *  depending on parseTarget.
 * - Default: (value) => void
 * - With parseTarget: ({ name, value }) => void
 */
export interface IInputFieldParseTarget<T = string | number | boolean> {
  name: string;
  value: T;
}

/** Return type of onInputChange of InputField */
export type InputFieldOnChange<T> =
  | ((value: T) => void)
  | ((evt: IInputFieldParseTarget<T>) => void);
