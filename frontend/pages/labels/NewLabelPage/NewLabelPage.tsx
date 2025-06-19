import React, { useCallback, useContext, useState } from "react";

import { QueryContext } from "context/query";
import useToggleSidePanel from "hooks/useToggleSidePanel";

import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";

import { RouteComponentProps } from "react-router";

const baseClass = "new-label-page";

interface INewLabelPageProps extends RouteComponentProps<never, never> {
  children: JSX.Element;
}

const NewLabelPage = ({ router, location, children }: INewLabelPageProps) => {
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(true);
  const [showOpenSidebarButton, setShowOpenSidebarButton] = useState(false);

  const onCloseSidebar = () => {
    setSidePanelOpen(false);
    setShowOpenSidebarButton(true);
  };

  const onOpenSidebar = () => {
    setSidePanelOpen(true);
    setShowOpenSidebarButton(false);
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  return (
    <>
      <MainContent className={baseClass}>
        <h1>Add label</h1>
        <p className={`${baseClass}__page-description`}>
          Dynamic (smart) labels are assigned to hosts if the query returns
          results. Manual labels are assigned to selected hosts.
        </p>
        {/* {React.cloneElement(children, {
          showOpenSidebarButton,
          onOpenSidebar,
          onOsqueryTableSelect,
        })} */}
        {/* TODO - replace this with the new aggregate form */}
      </MainContent>
      {isDynamicLabel && isSidePanelOpen && (
        <SidePanelContent>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={onCloseSidebar}
          />
        </SidePanelContent>
      )}
    </>
  );
};

export default NewLabelPage;
