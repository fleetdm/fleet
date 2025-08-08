import React, { useCallback } from "react";
import { useQueryClient } from "react-query";

import secretsAPI, { IListSecretsResponse } from "services/entities/secrets";
import { ISecret } from "interfaces/secrets";

import PaginatedList from "components/PaginatedList";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "secrets-batch-paginated-list";

export const SECRETS_PAGE_SIZE = 6;

const SecretsPaginatedList = () => {
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

  const renderSecretRow = (
    secret: ISecret,
    onChange: (secret: ISecret) => void
  ) => (
    <>
      <a>{secret.name}</a>
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
                <span>Add secret</span>
              </Button>
            </span>
          </div>
        }
      />
    </div>
  );
};

export default SecretsPaginatedList;
