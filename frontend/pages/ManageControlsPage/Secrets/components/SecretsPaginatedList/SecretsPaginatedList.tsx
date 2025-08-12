import React, { useCallback, useRef, useState } from "react";
import { useQueryClient } from "react-query";

import secretsAPI, { IListSecretsResponse } from "services/entities/secrets";
import { ISecret } from "interfaces/secrets";

import { stringToClipboard } from "utilities/copy_text";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import ListItem from "components/ListItem/ListItem";
import PaginatedList from "components/PaginatedList";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "secrets-batch-paginated-list";

export const SECRETS_PAGE_SIZE = 6;

const SecretsPaginatedList = () => {
  const [copyMessage, setCopyMessage] = useState("");
  const [copiedSecretName, setCopiedSecretName] = useState("");
  const copyMessageTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

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

  const onDeleteSecret = useCallback((secret: ISecret) => {
    // Logic to delete the secret
    console.log("Delete secret:", secret);
    // Here you would typically call an API to delete the secret
  }, []);

  const onAddSecret = useCallback(() => {
    console.log("ADD SECRET");
  }, []);

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
      />
    </div>
  );
};

export default SecretsPaginatedList;
