import PropTypes from "prop-types";
import { IFormField } from "./form_field";
import packInterface, { IPack } from "./pack";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  description: PropTypes.string,
  query: PropTypes.string,
  saved: PropTypes.bool,
  author_id: PropTypes.number,
  author_name: PropTypes.string,
  observer_can_run: PropTypes.bool,
  packs: PropTypes.arrayOf(packInterface),
});
export interface IQueryFormData {
  description?: string | number | boolean | any[];
  name?: string | number | boolean | any[];
  query?: string | number | boolean | any[];
  observer_can_run?: string | number | boolean | any[];
}

export interface IQuery {
  created_at: string;
  updated_at: string;
  id: number;
  name: string;
  description: string;
  query: string;
  saved: boolean;
  author_id: number;
  author_name: string;
  observer_can_run: boolean;
  packs: IPack[];
}

export interface IQueryFormFields {
  description: IFormField;
  name: IFormField;
  query: IFormField;
  observer_can_run: IFormField;
}
