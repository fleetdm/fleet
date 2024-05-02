// Note: This is a lite solution and does NOT include ability to
// "Select all across pages" nor to disable the header checkbox

import { HeaderProps, Row } from "react-table";

interface GetConditionalSelectHeaderCheckboxProps {
  /** react-table header props */
  headerProps: React.PropsWithChildren<HeaderProps<any>>;
  /** A function defining which rows are selectable */
  checkIfRowIsSelectable: (row: Row<any>) => boolean;
}

const getConditionalSelectHeaderCheckboxProps = ({
  headerProps,
  checkIfRowIsSelectable,
}: GetConditionalSelectHeaderCheckboxProps) => {
  // Define if the checkbox should show as checked or indeterminate
  const checkIfAllSelectableRowsSelected = (rows: Row<any>[]) =>
    rows.filter(checkIfRowIsSelectable).every((row) => row.isSelected);
  // Note: This is where we would include disabled logic if we needed it

  // Naming matches react-table v7 https://react-table-v7-docs.netlify.app/docs/api/useRowSelect#instance-properties
  // getToggleAllPageRowsSelectedProps: Function(props) => props
  const checked = checkIfAllSelectableRowsSelected(headerProps.rows);
  const indeterminate =
    !checked && headerProps.rows.some((row) => row.isSelected);

  const onChange = () => {
    // If all selectable rows are already selected, deselect all selectable rows on the page
    if (checkIfAllSelectableRowsSelected(headerProps.rows)) {
      headerProps.rows.forEach((row) => {
        headerProps.toggleRowSelected(row.id, false);
      });
    } else {
      // Otherwise select every selectable row on the page
      headerProps.page.forEach((row) => {
        const rowChecked = checkIfRowIsSelectable(row);
        headerProps.toggleRowSelected(row.id, rowChecked);
      });
    }
  };

  // Usual checkbox props
  const checkboxProps = headerProps.getToggleAllRowsSelectedProps();

  // For conditional select, we override value, indeterminate and onChange
  return {
    ...checkboxProps,
    value: checked,
    indeterminate,
    onChange,
    // disabled, // Not included
  };
};

export default { getConditionalSelectHeaderCheckboxProps };
