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
import EmptyTable from "components/EmptyTable";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import AddSecretModal from "./components/AddSecretModal";
import DeleteSecretModal from "./components/DeleteSecretModal";

const baseClass = "secrets-batch-paginated-list";

export const SECRETS_PAGE_SIZE = 20;

const Secrets = () => {
  const paginatedListRef = useRef<IPaginatedListHandle<ISecret>>(null);

  const [copyMessage, setCopyMessage] = useState("");
  const [copiedSecretName, setCopiedSecretName] = useState("");
  const copyMessageTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [secretToDelete, setSecretToDelete] = useState<ISecret | undefined>();
  const [showAddModal, setShowAddModal] = useState(false);
  const [pageNumber, setPageNumber] = useState(0);

  const { isGlobalAdmin, isGlobalMaintainer } = useContext(AppContext);

  const canEdit = isGlobalAdmin || isGlobalMaintainer;

  const apiParams = { page: pageNumber, per_page: SECRETS_PAGE_SIZE };
  const { data, isFetching: isLoading, refetch } = useQuery<
    IListSecretsResponse,
    Error,
    IListSecretsResponse
  >(["secrets", apiParams], () => secretsAPI.getSecrets(apiParams));

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
          variant="text-icon"
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

  if (isLoading) {
    return (
      <div className={`${baseClass}__loading`}>
        <Spinner />
      </div>
    );
  }
  if (data?.count === 0) {
    return (
      <>
        <EmptyTable
          header="No custom variables created yet"
          info="Add a custom variable to make it available in scripts and profiles."
          primaryButton={
            canEdit ? (
              <Button onClick={onClickAddSecret}>Add custom variable</Button>
            ) : undefined
          }
        />
        {showAddModal && (
          <AddSecretModal
            onCancel={() => setShowAddModal(false)}
            onSave={onSaveSecret}
          />
        )}
      </>
    );
  }
  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Manage custom variables that will be available in scripts and profiles.{" "}
        <CustomLink
          text="Learn more"
          url={`${FLEET_WEBSITE_URL}/guides/secrets-in-scripts-and-configuration-profiles`}
          newTab
        />
      </p>
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
            {canEdit && (
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <span>
                    <Button
                      variant="text-icon"
                      onClick={onClickAddSecret}
                      disabled={disableChildren}
                    >
                      <Icon name="plus" />
                      <span>Add custom variable</span>
                    </Button>
                  </span>
                )}
              />
            )}
          </div>
        }
        helpText={
          <span>
            Profiles can also use any of Fleet&rsquo;s{" "}
            <CustomLink
              url="https://fleetdm.com/docs/configuration/yaml-files#macos-settings-and-windows-settings"
              text="built-in variables"
              newTab
            />
          </span>
        }
      />
      {showAddModal && (
        <AddSecretModal
          onCancel={() => setShowAddModal(false)}
          onSave={onSaveSecret}
        />
      )}
      {showDeleteModal && (
        <DeleteSecretModal
          secret={secretToDelete}
          onExit={() => setShowDeleteModal(false)}
          onDeleteSecret={onDeleteSecret}
        />
      )}
    </div>
  );
};

export default Secrets;
