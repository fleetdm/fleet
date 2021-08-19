import React, { useState, useEffect } from "react";
import { connect, useDispatch } from "react-redux";
import { useQuery, useMutation } from "react-query";

// @ts-ignore
import Fleet from "fleet";
import {
  selectOsqueryTable, // @ts-ignore
} from "redux/nodes/components/QueryPages/actions";
import queryAPI from "services/entities/queries"; // @ts-ignore
import permissionUtils from "utilities/permissions";
import { IQueryFormData, IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";
import { IOsqueryTable } from "interfaces/osquery_table";
import { IUser } from "interfaces/user";
import { ICampaign } from "interfaces/campaign";

// @ts-ignore
import WarningBanner from "components/WarningBanner"; // @ts-ignore
import QuerySidePanel from "components/side_panels/QuerySidePanel"; // @ts-ignore
import QueryEditor from "pages/queries/QueryPage1/components/screens/QueryEditor";
import SelectTargets from "pages/queries/QueryPage1/components/screens/SelectTargets";
import RunQuery from "pages/queries/QueryPage1/components/screens/RunQuery";

interface IQueryPageProps {
  queryIdForEdit: string;
  selectedTargets: ITarget[];
  selectedOsqueryTable: IOsqueryTable;
  currentUser: IUser;
  isBasicTier: boolean;
}

const PAGE_STEP = {
  EDITOR: "EDITOR",
  TARGETS: "TARGETS",
  RUN: "RUN",
};

const baseClass = "query-page";

const QueryPage = ({
  queryIdForEdit,
  selectedTargets,
  selectedOsqueryTable,
  currentUser,
  isBasicTier,
}: IQueryPageProps) => {
  const dispatch = useDispatch();

  const [step, setStep] = useState<string>(PAGE_STEP.TARGETS);
  const [typedQueryBody, setTypedQueryBody] = useState<string>("");
  const [campaign, setCampaign] = useState<ICampaign | null>(null);
  const [queryIsRunning, setQueryIsRunning] = useState<boolean>(false);
  const [showQueryEditor, setShowQueryEditor] = useState<boolean>(false);
  const [liveQueryError, setLiveQueryError] = useState<string>("");

  const { status, data: storedQuery, error } = useQuery<IQuery, Error>(
    "query",
    () => queryAPI.load(queryIdForEdit),
    {
      enabled: !!queryIdForEdit,
    }
  );
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

    return false;
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

  const goToQueryEditor = () => {
    setStep(PAGE_STEP.EDITOR);
  };

  const goToSelectTargets = () => {
    setStep(PAGE_STEP.TARGETS);
  };

  const goToRunQuery = () => {
    setStep(PAGE_STEP.RUN);
  };

  const renderScreen = () => {
    const { TARGETS, RUN } = PAGE_STEP;

    switch (step) {
      case TARGETS:
        const step2Opts = {
          baseClass,
          typedQueryBody,
          selectedTargets: [...selectedTargets],
          campaign,
          isBasicTier,
          queryIdForEdit,
          goToQueryEditor,
          goToRunQuery,
          dispatch,
        };
        return <SelectTargets {...step2Opts} />;
      case RUN:
        const step3Opts = {
          baseClass,
          typedQueryBody,
          storedQuery,
          campaign,
          selectedTargets,
          queryIsRunning,
          setQueryIsRunning,
          setCampaign,
          dispatch,
        };
        return <RunQuery {...step3Opts} />;
      default:
        const step1Opts = {
          baseClass, 
          currentUser, 
          dispatch,
          storedQuery,
          createQuery,
          error,
          goToSelectTargets,
          setTypedQueryBody,
        };
        return <QueryEditor {...step1Opts} />;
    };
  };

  return (
    <div className={`${baseClass} has-sidebar`}>
      <div className={`${baseClass}__content`}>
        {renderScreen()}
        {renderLiveQueryWarning()}
      </div>
      <QuerySidePanel
        onOsqueryTableSelect={onOsqueryTableSelect}
        selectedOsqueryTable={selectedOsqueryTable}
      />
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
