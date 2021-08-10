import PropTypes from "prop-types";
import { IFormField } from "./form_field";

export default PropTypes.shape({
  description: PropTypes.string,
  name: PropTypes.string,
  query: PropTypes.string,
  id: PropTypes.number,
  interval: PropTypes.number,
  last_excuted: PropTypes.string,
  observer_can_run: PropTypes.bool,
  author_name: PropTypes.string,
  updated_at: PropTypes.string,
});
export interface IQueryFormData {
  description?: string | number | boolean | any[];
  name?: string | number | boolean | any[];
  query?: string | number | boolean | any[];
  observer_can_run?: string | number | boolean | any[];
};

export interface IQuery {
  description: string;
  name: string;
  query: string;
  id: number;
  interval: number;
  last_excuted: string;
  observer_can_run: boolean;
  author_name: string;
  updated_at: string;
};

export interface IQueryFormFields {
  description: IFormField;
  name: IFormField;
  query: IFormField;
  observer_can_run: IFormField;
};