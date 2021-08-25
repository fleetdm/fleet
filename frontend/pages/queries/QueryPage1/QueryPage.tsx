import React, { useState, useEffect } from "react";
import { connect, useDispatch } from "react-redux";
import { useQuery, useMutation } from "react-query";

// @ts-ignore
import Fleet from "fleet"; // @ts-ignore
import { selectOsqueryTable } from "redux/nodes/components/QueryPages/actions";
import {
  QUERIES_PAGE_STEPS,
  DEFAULT_QUERY,
} from "utilities/constants";
import queryAPI from "services/entities/queries"; // @ts-ignore
import permissionUtils from "utilities/permissions";
import { IQueryFormData, IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";
import { IOsqueryTable } from "interfaces/osquery_table";
import { IUser } from "interfaces/user";

// @ts-ignore
import WarningBanner from "components/WarningBanner"; // @ts-ignore
import QuerySidePanel from "components/side_panels/QuerySidePanel"; // @ts-ignore
import QueryEditor from "pages/queries/QueryPage1/screens/QueryEditor";
import SelectTargets from "pages/queries/QueryPage1/screens/SelectTargets";
import RunQuery from "pages/queries/QueryPage1/screens/RunQuery";

interface IQueryPageProps {
  queryIdForEdit: string;
  selectedTargets: ITarget[];
  selectedOsqueryTable: IOsqueryTable;
  currentUser: IUser;
  isBasicTier: boolean;
}

const baseClass = "query-page";

const QueryPage = ({
  queryIdForEdit,
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
  const [liveQueryError, setLiveQueryError] = useState<string>("");

  const { status, data: storedQuery = DEFAULT_QUERY, error } = useQuery<
    IQuery,
    Error
  >("query", () => queryAPI.load(queryIdForEdit), {
    enabled: !!queryIdForEdit,
  });
  const { mutateAsync: createQuery } = useMutation((formData: IQueryFormData) =>
    queryAPI.create(formData)
  );

  useEffect(() => {
    const checkLiveQuery = () => {
      Fleet.status.live_query().catch((response: any) => {
        try {
          const liveError = response.message.errors[0].reason;
          setLiveQueryError(liveError);
        } catch (e) {
          const liveError = `Unknown error: ${e}`;
          setLiveQueryError(liveError);
        }
      });
    };

    checkLiveQuery();
  }, []);

  const onOsqueryTableSelect = (tableName: string) => {
    dispatch(selectOsqueryTable(tableName));
  };

  const renderLiveQueryWarning = () => {
    if (!liveQueryError) {
      return false;
    }

    return (
      <WarningBanner className={`${baseClass}__warning`} shouldShowWarning>
        <h2 className={`${baseClass}__warning-title`}>
          Live query request failed
        </h2>
        <p>
          <span>Error:</span> {liveQueryError}
        </p>
      </WarningBanner>
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
      createQuery,
      error,
      onOsqueryTableSelect,
      goToSelectTargets: () => setStep(QUERIES_PAGE_STEPS[2]),
      setTypedQueryBody,
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
  return (
    <div className={`${baseClass} ${isFirstStep ? "has-sidebar" : ""}`}>
      <div className={`${baseClass}__content`}>
        {renderScreen()}
        {renderLiveQueryWarning()}
      </div>
      {isFirstStep && (
        <QuerySidePanel
          onOsqueryTableSelect={onOsqueryTableSelect}
          selectedOsqueryTable={selectedOsqueryTable}
        />
      )}
    </div>
  );
};

const mapStateToProps = (state: any, { params }: any) => {
  const { id: queryIdForEdit } = params;
  const { selectedOsqueryTable, selectedTargets } = state.components.QueryPages;
  const currentUser = state.auth.user;
  const config = state.app.config;
  const isBasicTier = permissionUtils.isBasicTier(config);

  return {
    queryIdForEdit,
    selectedTargets,
    selectedOsqueryTable,
    currentUser,
    isBasicTier,
  };
};

export default connect(mapStateToProps)(QueryPage);
