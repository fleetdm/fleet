import { HeaderProps, Row } from "react-table";

interface GetConditionalSelectHeaderCheckboxProps {
  /** react-table's header props */
  headerProps: React.PropsWithChildren<HeaderProps<any>> | any;
  /** A predicate - based on your business logic - to determine whether a given row should be selectable */
  checkIfRowIsSelectable: (row: Row<any>) => boolean;
  /** Whether to allow page selection. Default: true */
  shouldSelectPage?: boolean;
  value: any;
  indeterminate: any;
  onChange: () => any;
}

/**
 * A convenience method for react-table headers for allowing conditional select
 * @param headerProps react-table's header props
 * @param checkIfRowIsSelectable A predicate - based on your business logic - to determine whether a given row should be selectable
 * @param shouldSelectPage Whether to allow page selection. Default: true
 * @returns Modified `checkboxProps` to enforce the conditional select
 */

const getConditionalSelectHeaderCheckboxProps = ({
  headerProps,
  checkIfRowIsSelectable,
  shouldSelectPage = true,
}: GetConditionalSelectHeaderCheckboxProps) => {
  // Note that in my comments I differentiate between the standard logic and the logic for the conditional select
  const checkIfAllSelectableRowsSelected = (rows: Row<any>[]) =>
    rows.filter(checkIfRowIsSelectable).every((row) => row.isSelected);
  // Standard: Here we define the selection type for the next click: Select Page / Select All
  const isSelectPage =
    shouldSelectPage &&
    headerProps.page
      // For conditional select: Filter the rows based on your business logic
      .filter(checkIfRowIsSelectable)
      // Standard: `isSelectPage === true` if some of the rows are not yet selected
      // This (standard) logic might be confusing to understand at first, but - as a side note - the idea is as follows:
      // This is the variable that defines whether the header props that will be received FOR THE NEXT CLICK will be for Select Page or for Select All
      // Try to play this out in your head:
      //  - Initially, none of the rows are selected, so when we clicking the button initially, we will select only the (selectable) rows on the page (i.e. Select Page), hence the next click will be for Select All, hence `isSelectPage` will be `false`
      //  - When clicking again, we will select the rest of the (selectable) rows (i.e. Select All). The next click will again be Select All (for de-selecting all), hence `isSelectPage` will be `false`
      //  - Finally, when clicking again, we will de-select all rows. The next click will be for Select Page, hence `isSelectPage` will `true`
      .some((row: any) => !row.isSelected);

  // Standard: Get the props based on Select Page / Select All
  const checkboxProps = isSelectPage
    ? headerProps.getToggleAllPageRowsSelectedProps()
    : headerProps.getToggleAllRowsSelectedProps();

  // For conditional select: The header checkbox should be:
  //   - checked if all selectable rows are selected
  //   - indeterminate if only some selectable rows are selected (but not all)
  const disabled = headerProps.rows.filter(checkIfRowIsSelectable).length === 0;
  const checked =
    !disabled && checkIfAllSelectableRowsSelected(headerProps.rows);
  const indeterminate =
    !checked && headerProps.rows.some((row: any) => row.isSelected);

  // For conditional select: This is where the magic happens
  const onChange = () => {
    // If we're in Select All and all selectable rows are already selected: deselect all rows
    if (!isSelectPage && checkIfAllSelectableRowsSelected(headerProps.rows)) {
      headerProps.rows.forEach((row: any) => {
        headerProps.toggleRowSelected(row.id, false);
      });
    } else {
      // Otherwise:
      // First, define the rows to work with: if we're in Select Page, use `headerProps.page`, otherwise (Select All) use headerProps.rows
      const rows = isSelectPage ? headerProps.page : headerProps.rows;
      // Then select every selectable row
      rows.forEach((row: any) => {
        const rowChecked = checkIfRowIsSelectable(row);
        headerProps.toggleRowSelected(row.id, rowChecked);
      });
    }
  };

  // For conditional select: override checked, indeterminate and onChange - to enforce conditional select based on our business logic
  return {
    ...checkboxProps,
    checked,
    indeterminate,
    onChange,
    disabled,
  };
};

export default getConditionalSelectHeaderCheckboxProps;
