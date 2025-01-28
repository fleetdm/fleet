// from https://stackoverflow.com/a/68213902/15458245

import { HeaderProps, Row } from "react-table";

interface GetConditionalSelectHeaderCheckboxProps {
  /** react-table header props */
  headerProps: React.PropsWithChildren<HeaderProps<any>>;
  checkIfRowIsSelectable: (row: Row<any>) => boolean;
}

export const getConditionalSelectHeaderCheckboxProps = ({
  headerProps,
  checkIfRowIsSelectable,
}: GetConditionalSelectHeaderCheckboxProps) => {
  // Define if the checkbox should show as checked or indeterminate
  const checkIfAllSelectableRowsSelected = (rows: Row<any>[]) =>
    rows.filter(checkIfRowIsSelectable).every((row) => row.isSelected);
  const allSelectableRowsSelected = checkIfAllSelectableRowsSelected(
    headerProps.rows
  );
  const indeterminate =
    !allSelectableRowsSelected &&
    headerProps.rows.some((row) => row.isSelected);

  const onChange = () => {
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

  return {
    ...checkboxProps,
    value: allSelectableRowsSelected,
    indeterminate,
    onChange,
  };
};

export default { getConditionalSelectHeaderCheckboxProps };
