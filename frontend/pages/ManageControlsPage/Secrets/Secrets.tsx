import React, { useCallback, useRef, useState } from "react";
import { InjectedRouter } from "react-router";

import { useQueryClient } from "react-query";

import secretsAPI, { IListSecretsResponse } from "services/entities/secrets";
import { ISecret, ISecretPayload } from "interfaces/secrets";

import { stringToClipboard } from "utilities/copy_text";
import CustomLink from "components/CustomLink";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import ListItem from "components/ListItem/ListItem";
import PaginatedList, { IPaginatedListHandle } from "components/PaginatedList";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import AddSecretModal from "./components/AddSecretModal";
import DeleteSecretModal from "./components/DeleteSecretModal";
import { set } from "lodash";

const baseClass = "secrets-batch-paginated-list";

export const SECRETS_PAGE_SIZE = 6;

interface ISecretsProps {
  router: InjectedRouter; // v3
  teamIdForApi: number;
  currentPage: number;
}

const Secrets = ({ router, currentPage, teamIdForApi }: ISecretsProps) => {
  const paginatedListRef = useRef<IPaginatedListHandle<ISecret>>(null);

  const [copyMessage, setCopyMessage] = useState("");
  const [copiedSecretName, setCopiedSecretName] = useState("");
  const copyMessageTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [secretToDelete, setSecretToDelete] = useState<ISecret | undefined>();
  const [showAddModal, setShowAddModal] = useState(false);

  // Fetch a single page of scripts.
  const queryClient = useQueryClient();

  const fetchPage = useCallback(
    (pageNumber: number) => {
      // scripts not supported for All teams
      const fetchPromise = queryClient.fetchQuery(
        [
          {
            page: pageNumber,
            per_page: SECRETS_PAGE_SIZE,
          },
        ],
        ({ queryKey }) => {
          return secretsAPI.getSecrets(queryKey[0]);
        }
      );

      return fetchPromise.then(({ secrets }: IListSecretsResponse) => {
        return secrets || [];
      });
    },
    [queryClient]
  );

  const onAddSecret = () => {
    setShowAddModal(true);
  };

  const onSave = (secretName: string, secretValue: string): Promise<object> => {
    const newSecret: ISecretPayload = {
      name: secretName,
      value: secretValue,
    };
    const addSecretPromise = secretsAPI.addSecret(newSecret);
    // Handle success by closing the modal and reloading the list.
    addSecretPromise.then(() => {
      setShowAddModal(false);
      paginatedListRef.current?.reload({ keepPage: true });
    });
    // The modal will handle errors and display appropriate messages.
    return addSecretPromise;
  };

  const onDeleteSecret = (secret: ISecret) => {
    setSecretToDelete(secret);
    setShowDeleteModal(true);
  };

  const onDelete = useCallback(() => {
    if (!secretToDelete) {
      return;
    }
    secretsAPI
      .deleteSecret(secretToDelete.id)
      .then(() => {
        paginatedListRef.current?.reload({ keepPage: true });
        setShowDeleteModal(false);
      })
      .catch((error) => {
        console.error("Error deleting secret:", error);
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
          onDeleteSecret(secret);
        }}
      >
        <>
          <Icon name="trash" color="ui-fleet-black-75" />
        </>
      </Button>
    </>
  );

  return (
    <div className={`${baseClass}`}>
      <PaginatedList<ISecret>
        ref={paginatedListRef}
        renderItemRow={renderSecretRow}
        count={2}
        fetchPage={fetchPage}
        onClickRow={(secret) => secret}
        heading={
          <div className={`${baseClass}__header`}>
            <span>Custom variables</span>
            <span>
              <Button variant="text-icon" onClick={onAddSecret}>
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
            return onSave(secretName, secretValue);
          }}
        />
      )}
      {showDeleteModal && (
        <DeleteSecretModal
          secret={secretToDelete}
          onCancel={() => setShowDeleteModal(false)}
          onDelete={onDelete}
        />
      )}
    </div>
  );
};

export default Secrets;
