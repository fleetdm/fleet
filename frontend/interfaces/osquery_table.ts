import PropTypes from "prop-types";

export default PropTypes.shape({
  columns: PropTypes.arrayOf(
    PropTypes.shape({
      description: PropTypes.string,
      name: PropTypes.string,
      type: PropTypes.string,
    })
  ),
  description: PropTypes.string,
  name: PropTypes.string,
  platform: PropTypes.string,
});

interface ITableColumn {
  description: string;
  name: string;
  type: string;
}

export interface IOsqueryTable {
  columns: ITableColumn[];
  description: string;
  name: string;
  platform: string;
}
