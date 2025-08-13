import React, {
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

import { useQueryClient } from "react-query";

import { NotificationContext } from "context/notification";
import secretsAPI, { IListSecretsResponse } from "services/entities/secrets";
import { ISecret, ISecretPayload } from "interfaces/secrets";

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
  const { renderFlash } = useContext(NotificationContext);

  const paginatedListRef = useRef<IPaginatedListHandle<ISecret>>(null);

  const [copyMessage, setCopyMessage] = useState("");
  const [copiedSecretName, setCopiedSecretName] = useState("");
  const copyMessageTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [secretToDelete, setSecretToDelete] = useState<ISecret | undefined>();
  const [showAddModal, setShowAddModal] = useState(false);

  const [count, setCount] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSaving, setIsSaving] = useState<boolean>(false);
  const [isDeleting, setIsDeleting] = useState<boolean>(false);

  const queryClient = useQueryClient();

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

  // Allow the list to fetch the count of secrets.
  const fetchCount = useCallback(() => {
    if (count !== null) {
      return Promise.resolve(count);
    }
    return Promise.resolve(0);
  }, [count]);

  const onClickAddSecret = () => {
    setShowAddModal(true);
  };

  const onSaveSecret = (
    secretName: string,
    secretValue: string
  ): Promise<object> => {
    setIsSaving(true);
    const newSecret: ISecretPayload = {
      name: secretName,
      value: secretValue,
    };
    const addSecretPromise = secretsAPI.addSecret(newSecret);
    // Handle success by closing the modal and reloading the list.
    addSecretPromise
      .then(() => {
        setShowAddModal(false);
        // If we're showing a list right now, tell it to reload.
        if (paginatedListRef.current) {
          paginatedListRef.current?.reload();
        } else {
          // If we were showing the empty state, fetch the first page
          // to populate the list.
          fetchPage(0);
        }
      })
      .catch((error) => {
        if (error.status !== 409) {
          renderFlash(
            "error",
            "An error occurred while saving the secret. Please try again."
          );
        }
      })
      .finally(() => {
        setIsSaving(false);
      });
    // The modal will handle conflict errors and display appropriate messages.
    return addSecretPromise;
  };

  const onClickDeleteSecret = (secret: ISecret) => {
    setSecretToDelete(secret);
    setShowDeleteModal(true);
  };

  const onDeleteSecret = useCallback(() => {
    if (!secretToDelete) {
      return;
    }
    setIsDeleting(true);
    secretsAPI
      .deleteSecret(secretToDelete.id)
      .then(() => {
        paginatedListRef.current?.reload();
        setShowDeleteModal(false);
      })
      .catch((error) => {
        console.error("Error deleting secret:", error);
      })
      .finally(() => {
        setIsDeleting(false);
      });
  }, [secretToDelete]);

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
            <Button onClick={onClickAddSecret}>Add custom variable</Button>
          }
        />
        {showAddModal && (
          <AddSecretModal
            onCancel={() => setShowAddModal(false)}
            onSubmit={(secretName: string, secretValue: string) => {
              return onSaveSecret(secretName, secretValue);
            }}
            isSaving={isSaving}
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
        fetchCount={fetchCount}
        fetchPage={fetchPage}
        onClickRow={(secret) => secret}
        heading={
          <div className={`${baseClass}__header`}>
            <span>Custom variables</span>
            <span>
              <Button variant="text-icon" onClick={onClickAddSecret}>
                <Icon name="plus" />
                <span>Add custom variable</span>
              </Button>
            </span>
          </div>
        }
        helpText={
          <span>
            Profiles can also use any of Fleet&rsquo;s{" "}
            <CustomLink url="#" text="built-in variables" newTab />
          </span>
        }
      />
      {showAddModal && (
        <AddSecretModal
          onCancel={() => setShowAddModal(false)}
          onSubmit={(secretName: string, secretValue: string) => {
            return onSaveSecret(secretName, secretValue);
          }}
          isSaving={isSaving}
        />
      )}
      {showDeleteModal && (
        <DeleteSecretModal
          secret={secretToDelete}
          onCancel={() => setShowDeleteModal(false)}
          onDelete={onDeleteSecret}
          isDeleting={isDeleting}
        />
      )}
    </div>
  );
};

export default Secrets;
