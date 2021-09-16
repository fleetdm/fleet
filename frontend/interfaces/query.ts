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
  scheduled_query_stats: PropTypes.shape({
    total_user_time: PropTypes.number,
    total_system_time: PropTypes.number,
    executions: PropTypes.number,
  }),
});
export interface IQueryFormData {
  description?: string | number | boolean | any[] | undefined;
  name?: string | number | boolean | any[] | undefined;
  query?: string | number | boolean | any[] | undefined;
  observer_can_run?: string | number | boolean | any[] | undefined;
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
  scheduled_query_stats: {
    total_user_time: number;
    total_system_time: number;
    executions: number;
  };
}

export interface IQueryFormFields {
  description: IFormField;
  name: IFormField;
  query: IFormField;
  observer_can_run: IFormField;
}
