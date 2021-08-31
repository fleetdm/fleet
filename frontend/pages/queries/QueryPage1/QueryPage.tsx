import React, { useState, useEffect } from "react";
import { connect, useDispatch } from "react-redux";
import { useQuery, useMutation } from "react-query";
import { Params } from "react-router/lib/Router";

// @ts-ignore
import Fleet from "fleet"; // @ts-ignore
import { selectOsqueryTable } from "redux/nodes/components/QueryPages/actions";
import { QUERIES_PAGE_STEPS, DEFAULT_QUERY } from "utilities/constants";
import queryAPI from "services/entities/queries"; // @ts-ignore
import permissionUtils from "utilities/permissions";
import { IQueryFormData, IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";
import { IOsqueryTable } from "interfaces/osquery_table";
import { IUser } from "interfaces/user";

// @ts-ignore
// import WarningBanner from "components/WarningBanner";
import QuerySidePanel from "components/side_panels/QuerySidePanel"; // @ts-ignore
import QueryEditor from "pages/queries/QueryPage1/screens/QueryEditor";
import SelectTargets from "pages/queries/QueryPage1/screens/SelectTargets";
import RunQuery from "pages/queries/QueryPage1/screens/RunQuery";
import ExternalURLIcon from "../../../../assets/images/icon-external-url-12x12@2x.png";

interface IQueryPageProps {
  params: Params;
  queryIdForEdit: string;
  selectedTargets: ITarget[];
  selectedOsqueryTable: IOsqueryTable;
  currentUser: IUser;
  isBasicTier: boolean;
}

interface IStoredQueryResponse {
  query: IQuery;
} 

const baseClass = "query-page1";

const QueryPage = ({
  params: { id: queryIdForEdit },
  selectedTargets,
  selectedOsqueryTable,
  currentUser,
  isBasicTier,
}: IQueryPageProps) => {
  const dispatch = useDispatch();

  const [step, setStep] = useState<string>(QUERIES_PAGE_STEPS[1]);
  const [typedQueryBody, setTypedQueryBody] = useState<string>(
    DEFAULT_QUERY.query
  );
  const [queryIsRunning, setQueryIsRunning] = useState<boolean>(false);
  const [showQueryEditor, setShowQueryEditor] = useState<boolean>(false);
  const [isLiveQueryRunnable, setIsLiveQueryRunnable] = useState<boolean>(true);
  const [isSidebarOpen, setIsSidebarOpen] = useState<boolean>(true);
  const [
    showOpenSchemaActionText,
    setShowOpenSchemaActionText,
  ] = useState<boolean>(false);

  const { 
    isLoading: isStoredQueryLoading, 
    data: storedQuery = DEFAULT_QUERY, 
    error: storedQueryError
  } = useQuery<IStoredQueryResponse, Error, IQuery>(
    "query", 
    () => queryAPI.load(queryIdForEdit), 
    {
      enabled: !!queryIdForEdit,
      select: (data: IStoredQueryResponse) => data.query
    }
  );
  
  const { mutateAsync: createQuery } = useMutation((formData: IQueryFormData) =>
    queryAPI.create(formData)
  );

  useEffect(() => {
    const detectIsFleetQueryRunnable = () => {
      Fleet.status.live_query().catch(() => {
        setIsLiveQueryRunnable(false);
      });
    };

    detectIsFleetQueryRunnable();
  }, []);

  useEffect(() => {
    setShowOpenSchemaActionText(!isSidebarOpen);
  }, [isSidebarOpen]);

  const onOsqueryTableSelect = (tableName: string) => {
    dispatch(selectOsqueryTable(tableName));
  };

  const onCloseSidebar = () => {
    setIsSidebarOpen(false);
  };

  const onOpenSchemaSidebar = () => {
    setIsSidebarOpen(true);
  };

  const renderLiveQueryWarning = (): JSX.Element | null => {
    if (isLiveQueryRunnable) {
      return null;
    }

    return (
      <div className={`${baseClass}__warning`}>
        <div className={`${baseClass}__message`}>
          <p>
            Fleet is unable to run a live query. Refresh the page or log in again. 
            If this keeps happening please <a target="_blank" rel="noopener noreferrer" href="https://github.com/fleetdm/fleet/issues/new/choose">file an issue <img alt="" src={ExternalURLIcon} /></a>
          </p>
        </div>
      </div>
    );
  };

  const renderScreen = () => {
    const commonOpts = {
      baseClass,
      dispatch,
    };

    const step1Opts = {
      ...commonOpts,
      currentUser,
      storedQuery,
      showOpenSchemaActionText,
      isStoredQueryLoading,
      error: storedQueryError,
      createQuery,
      onOsqueryTableSelect,
      goToSelectTargets: () => setStep(QUERIES_PAGE_STEPS[2]),
      setTypedQueryBody,
      onOpenSchemaSidebar,
      renderLiveQueryWarning,
    };

    const step2Opts = {
      ...commonOpts,
      selectedTargets: [...selectedTargets],
      isBasicTier,
      queryIdForEdit,
      goToQueryEditor: () => setStep(QUERIES_PAGE_STEPS[1]),
      goToRunQuery: () => setStep(QUERIES_PAGE_STEPS[3]),
    };

    const step3Opts = {
      ...commonOpts,
      typedQueryBody,
      storedQuery,
      selectedTargets,
    };

    switch (step) {
      case QUERIES_PAGE_STEPS[2]:
        return <SelectTargets {...step2Opts} />;
      case QUERIES_PAGE_STEPS[3]:
        return <RunQuery {...step3Opts} />;
      default:
        return <QueryEditor {...step1Opts} />;
    }
  };

  const isFirstStep = step === QUERIES_PAGE_STEPS[1];
  const sidebarClass = isFirstStep && isSidebarOpen && "has-sidebar";
  return (
    <div className={`${baseClass} ${sidebarClass}`}>
      <div className={`${baseClass}__content`}>
        {renderScreen()}
      </div>
      {isFirstStep && isSidebarOpen && (
        <QuerySidePanel
          onOsqueryTableSelect={onOsqueryTableSelect}
          selectedOsqueryTable={selectedOsqueryTable}
          onClose={onCloseSidebar}
        />
      )}
    </div>
  );
};

const mapStateToProps = (state: any, { params }: any) => {
  // const { id: queryIdForEdit } = params;
  const { selectedOsqueryTable, selectedTargets } = state.components.QueryPages;
  const currentUser = state.auth.user;
  const config = state.app.config;
  const isBasicTier = permissionUtils.isBasicTier(config);

  return {
    // queryIdForEdit,
    selectedTargets,
    selectedOsqueryTable,
    currentUser,
    isBasicTier,
  };
};

export default connect(mapStateToProps)(QueryPage);
