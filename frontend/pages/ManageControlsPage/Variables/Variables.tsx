import React, { useContext, useEffect, useRef, useState } from "react";

import { useQuery } from "react-query";

import secretsAPI, { IListSecretsResponse } from "services/entities/secrets";
import { ISecret } from "interfaces/secrets";

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

export const SECRETS_PAGE_SIZE = 20;

const Variables = () => {
  const paginatedListRef = useRef<IPaginatedListHandle<ISecret>>(null);

  const [copyMessage, setCopyMessage] = useState("");
  const [copiedSecretName, setCopiedSecretName] = useState("");
  const copyMessageTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [secretToDelete, setSecretToDelete] = useState<ISecret | undefined>();
  const [showAddModal, setShowAddModal] = useState(false);
  const [pageNumber, setPageNumber] = useState(0);

  const { isGlobalAdmin, isGlobalMaintainer, isPremiumTier } = useContext(
    AppContext
  );

  const canEdit = isGlobalAdmin || isGlobalMaintainer;

  const apiParams = { page: pageNumber, per_page: SECRETS_PAGE_SIZE };
  const { data, isFetching: isLoading, refetch } = useQuery<
    IListSecretsResponse,
    Error,
    IListSecretsResponse
  >(["secrets", apiParams], () => secretsAPI.getSecrets(apiParams));

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

  const onClickAddSecret = () => {
    setShowAddModal(true);
  };

  const onSaveSecret = () => {
    setShowAddModal(false);
    refetch();
  };

  const onDeleteSecret = () => {
    setShowDeleteModal(false);
    refetch();
  };

  const onClickDeleteSecret = (secret: ISecret) => {
    setSecretToDelete(secret);
    setShowDeleteModal(true);
  };

  const getTokenFromSecretName = (secretName: string): string => {
    return `$FLEET_SECRET_${secretName.toUpperCase()}`;
  };

  const onCopySecretName = (evt: React.MouseEvent, secretName: string) => {
    evt.preventDefault();

    if (copyMessageTimeoutIdRef.current) {
      clearTimeout(copyMessageTimeoutIdRef.current);
    }

    setCopiedSecretName(secretName);
    stringToClipboard(getTokenFromSecretName(secretName))
      .then(() => setCopyMessage("Copied!"))
      .catch(() => setCopyMessage("Copy failed"));

    // Clear message after 1 second
    copyMessageTimeoutIdRef.current = setTimeout(() => {
      setCopyMessage("");
      setCopiedSecretName("");
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

  const renderSecretRow = (secret: ISecret) => (
    <>
      <ListItem
        title={secret.name.toUpperCase()}
        details={
          <span>
            <span className="secret-details__text">
              Updated{" "}
              <HumanTimeDiffWithDateTip timeString={secret.updated_at} /> &bull;{" "}
              {getTokenFromSecretName(secret.name)}
            </span>
            <Button
              variant="unstyled"
              className={`${baseClass}__copy-secret-icon`}
              onClick={(e: React.MouseEvent<HTMLButtonElement>) =>
                onCopySecretName(e, secret.name)
              }
            >
              <Icon name="copy" />
            </Button>
            {copyMessage && copiedSecretName === secret.name && (
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
            onClickDeleteSecret(secret);
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
                  <Button onClick={onClickAddSecret} disabled={disableChildren}>
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
            onSave={onSaveSecret}
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
                onClick={onClickAddSecret}
                disabled={disableChildren}
              >
                <Icon name="plus" />
                <span>Add custom variable</span>
              </Button>
            )}
          />
        )}
      </div>
      <PaginatedList<ISecret>
        ref={paginatedListRef}
        pageSize={SECRETS_PAGE_SIZE}
        renderItemRow={renderSecretRow}
        count={data?.count || 0}
        data={data?.custom_variables || []}
        currentPage={pageNumber}
        onChangePage={setPageNumber}
        onClickRow={(secret) => secret}
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
          onSave={onSaveSecret}
        />
      )}
      {showDeleteModal && (
        <DeleteCustomVariableModal
          secret={secretToDelete}
          onExit={() => setShowDeleteModal(false)}
          onDeleteSecret={onDeleteSecret}
        />
      )}
    </div>
  );
};

export default Variables;
