import React, { useContext, useEffect, useRef, useState } from "react";

import { useQuery } from "react-query";

import variablesAPI, {
  IListVariablesResponse,
} from "services/entities/variables";
import { IVariable } from "interfaces/variables";

import { AppContext } from "context/app";

import { stringToClipboard } from "utilities/copy_text";
import { FLEET_WEBSITE_URL } from "utilities/constants";
import CustomLink from "components/CustomLink";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import ListItem from "components/ListItem/ListItem";
import PaginatedList, { IPaginatedListHandle } from "components/PaginatedList";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import PageDescription from "components/PageDescription";
import AddCustomVariableModal from "./components/AddCustomVariableModal";
import DeleteCustomVariableModal from "./components/DeleteCustomVariableModal";

const baseClass = "variables";

export const VARIABLES_PAGE_SIZE = 20;

const Variables = () => {
  const paginatedListRef = useRef<IPaginatedListHandle<IVariable>>(null);

  const [copyMessage, setCopyMessage] = useState("");
  const [copiedVariableName, setCopiedVariableName] = useState("");
  const copyMessageTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [variableToDelete, setVariableToDelete] = useState<
    IVariable | undefined
  >();
  const [showAddModal, setShowAddModal] = useState(false);
  const [pageNumber, setPageNumber] = useState(0);

  const { isGlobalAdmin, isGlobalMaintainer, isPremiumTier } = useContext(
    AppContext
  );

  const canEdit = isGlobalAdmin || isGlobalMaintainer;

  const apiParams = { page: pageNumber, per_page: VARIABLES_PAGE_SIZE };
  const { data, isFetching: isLoading, refetch } = useQuery<
    IListVariablesResponse,
    Error,
    IListVariablesResponse
  >(["variables", apiParams], () => variablesAPI.getVariables(apiParams));

  // Open add modal via query param (e.g. from command palette)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("add_variable") === "1") {
      setShowAddModal(true);
      params.delete("add_variable");
      const qs = params.toString();
      window.history.replaceState(
        {},
        "",
        qs ? `${window.location.pathname}?${qs}` : window.location.pathname
      );
    }
  }, []);

  const onClickAddVariable = () => {
    setShowAddModal(true);
  };

  const onSaveVariable = () => {
    setShowAddModal(false);
    refetch();
  };

  const onDeleteVariable = () => {
    setShowDeleteModal(false);
    refetch();
  };

  const onClickDeleteVariable = (variable: IVariable) => {
    setVariableToDelete(variable);
    setShowDeleteModal(true);
  };

  const getTokenFromVariableName = (variableName: string): string => {
    return `$FLEET_SECRET_${variableName.toUpperCase()}`;
  };

  const onCopyVariableName = (evt: React.MouseEvent, variableName: string) => {
    evt.preventDefault();

    if (copyMessageTimeoutIdRef.current) {
      clearTimeout(copyMessageTimeoutIdRef.current);
    }

    setCopiedVariableName(variableName);
    stringToClipboard(getTokenFromVariableName(variableName))
      .then(() => setCopyMessage("Copied!"))
      .catch(() => setCopyMessage("Copy failed"));

    // Clear message after 1 second
    copyMessageTimeoutIdRef.current = setTimeout(() => {
      setCopyMessage("");
      setCopiedVariableName("");
    }, 1000);

    return false;
  };

  // Cleanup timeout on unmount.
  useEffect(() => {
    return () => {
      if (copyMessageTimeoutIdRef.current) {
        clearTimeout(copyMessageTimeoutIdRef.current);
      }
    };
  }, []);

  const renderVariableRow = (variable: IVariable) => (
    <>
      <ListItem
        title={variable.name.toUpperCase()}
        details={
          <span>
            <span className="variable-details__text">
              Updated{" "}
              <HumanTimeDiffWithDateTip timeString={variable.updated_at} />{" "}
              &bull; {getTokenFromVariableName(variable.name)}
            </span>
            <Button
              variant="unstyled"
              className={`${baseClass}__copy-variable-icon`}
              onClick={(e: React.MouseEvent<HTMLButtonElement>) =>
                onCopyVariableName(e, variable.name)
              }
            >
              <Icon name="copy" />
            </Button>
            {copyMessage && copiedVariableName === variable.name && (
              <span
                className={`${baseClass}__copy-message`}
              >{`${copyMessage} `}</span>
            )}
          </span>
        }
      />
      {canEdit && (
        <Button
          variant="icon"
          onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
            e.stopPropagation();
            onClickDeleteVariable(variable);
          }}
        >
          <>
            <Icon name="trash" color="ui-fleet-black-75" />
          </>
        </Button>
      )}
    </>
  );

  const renderPageDescription = () => (
    <PageDescription
      variant="tab-panel"
      content={
        <>
          {isPremiumTier
            ? "Manage custom variables that will be available in scripts and profiles across all fleets."
            : "Manage custom variables that will be available in scripts and profiles."}{" "}
          <CustomLink
            text="Learn more"
            url={`${FLEET_WEBSITE_URL}/guides/secrets-in-scripts-and-configuration-profiles`}
            newTab
          />
        </>
      }
    />
  );

  if (isLoading) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__page-header`}>
          {renderPageDescription()}
        </div>
        <div className={`${baseClass}__loading`}>
          <Spinner />
        </div>
      </div>
    );
  }

  if (data?.count === 0) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__page-header`}>
          {renderPageDescription()}
        </div>
        <EmptyState
          variant="header-list"
          header="No custom variables"
          info={
            canEdit
              ? "Add a custom variable to make it available in scripts and profiles."
              : "No custom variables are available for scripts and profiles."
          }
          primaryButton={
            canEdit ? (
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <Button
                    onClick={onClickAddVariable}
                    disabled={disableChildren}
                  >
                    Add custom variable
                  </Button>
                )}
              />
            ) : undefined
          }
        />
        {showAddModal && (
          <AddCustomVariableModal
            onCancel={() => setShowAddModal(false)}
            onSave={onSaveVariable}
          />
        )}
      </div>
    );
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__page-header`}>
        {renderPageDescription()}
        {canEdit && (
          <GitOpsModeTooltipWrapper
            renderChildren={(disableChildren) => (
              <Button
                variant="inverse"
                size="small"
                onClick={onClickAddVariable}
                disabled={disableChildren}
              >
                <Icon name="plus" />
                <span>Add custom variable</span>
              </Button>
            )}
          />
        )}
      </div>
      <PaginatedList<IVariable>
        ref={paginatedListRef}
        pageSize={VARIABLES_PAGE_SIZE}
        renderItemRow={renderVariableRow}
        count={data?.count || 0}
        data={data?.custom_variables || []}
        currentPage={pageNumber}
        onChangePage={setPageNumber}
        onClickRow={(variable) => variable}
        heading={
          <div className={`${baseClass}__header`}>
            <span>Custom variables</span>
          </div>
        }
        helpText={
          <span>
            Profiles can also use any of Fleet&rsquo;s{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/built-in-variables"
              text="built-in variables"
              newTab
            />
          </span>
        }
      />
      {showAddModal && (
        <AddCustomVariableModal
          onCancel={() => setShowAddModal(false)}
          onSave={onSaveVariable}
        />
      )}
      {showDeleteModal && (
        <DeleteCustomVariableModal
          variable={variableToDelete}
          onExit={() => setShowDeleteModal(false)}
          onDeleteVariable={onDeleteVariable}
        />
      )}
    </div>
  );
};

export default Variables;
