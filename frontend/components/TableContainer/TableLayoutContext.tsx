import { createContext } from "react";

// Signals to descendants that they're rendered inside a TableContainer's
// data-table block — which has overflow-x: auto on its wrapper. Popup
// components (e.g. ActionsDropdown) read this to decide whether to portal
// their menu to document.body so the wrapper's overflow doesn't clip it.
const TableLayoutContext = createContext<{ insideTable: boolean }>({
  insideTable: false,
});

export default TableLayoutContext;
