import React, {
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

import { useQueryClient } from "react-query";

import secretsAPI, { IListSecretsResponse } from "services/entities/secrets";
import { ISecret } from "interfaces/secrets";

import { AppContext } from "context/app";

import { stringToClipboard } from "utilities/copy_text";
import CustomLink from "components/CustomLink";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import ListItem from "components/ListItem/ListItem";
import PaginatedList, { IPaginatedListHandle } from "components/PaginatedList";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import EmptyTable from "components/EmptyTable";
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

  const [count, setCount] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  const queryClient = useQueryClient();

  const {
    isGlobalAdmin,
    isTeamAdmin,
    isGlobalMaintainer,
    isTeamMaintainer,
  } = useContext(AppContext);

  const canEdit =
    isGlobalAdmin || isTeamAdmin || isGlobalMaintainer || isTeamMaintainer;

  // Fetch a single page of secrets.
  const fetchPage = useCallback(
    (pageNumber: number) => {
      setIsLoading(true);
      const fetchPromise = queryClient.fetchQuery(
        [
          {
            page: pageNumber,
            per_page: SECRETS_PAGE_SIZE,
          },
        ],
        ({ queryKey }) => {
          return secretsAPI.getSecrets(queryKey[0]);
        },
        {
          staleTime: 100,
        }
      );

      return fetchPromise.then(
        ({
          custom_variables: secrets,
          count: numSecrets,
        }: IListSecretsResponse) => {
          setCount(numSecrets);
          setIsLoading(false);
          return secrets || [];
        }
      );
    },
    [queryClient]
  );

  const onClickAddSecret = () => {
    setShowAddModal(true);
  };

  const onSaveSecret = () => {
    setShowAddModal(false);
    // If we're showing a list right now, tell it to reload.
    if (paginatedListRef.current) {
      paginatedListRef.current?.reload();
    } else {
      // If we were showing the empty state, fetch the first page
      // to populate the list.
      fetchPage(0);
    }
  };

  const onClickDeleteSecret = (secret: ISecret) => {
    setSecretToDelete(secret);
    setShowDeleteModal(true);
  };

  const onDeleteSecret = () => {
    paginatedListRef.current?.reload();
    setShowDeleteModal(false);
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
            Updated <HumanTimeDiffWithDateTip timeString={secret.updated_at} />{" "}
            &bull; {getTokenFromSecretName(secret.name)}
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

  if (count === null && !isLoading) {
    fetchPage(0);
  }
  if (isLoading && count === null) {
    return (
      <div className={`${baseClass}__loading`}>
        <Spinner />
      </div>
    );
  }
  if (count === 0) {
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
      <p>
        Manage custom variables that will be available in scripts and profiles.
      </p>
      <PaginatedList<ISecret>
        ref={paginatedListRef}
        pageSize={SECRETS_PAGE_SIZE}
        renderItemRow={renderSecretRow}
        count={count || 0}
        fetchPage={fetchPage}
        onClickRow={(secret) => secret}
        heading={
          <div className={`${baseClass}__header`}>
            <span>Custom variables</span>
            {canEdit && (
              <span>
                <Button variant="text-icon" onClick={onClickAddSecret}>
                  <Icon name="plus" />
                  <span>Add custom variable</span>
                </Button>
              </span>
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
          onCancel={() => setShowDeleteModal(false)}
          onDelete={onDeleteSecret}
        />
      )}
    </div>
  );
};

export default Secrets;
