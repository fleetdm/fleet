import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import { getPathWithQueryParams } from "utilities/url";

import { ICreateQueryRequestBody } from "interfaces/schedulable_query";

import queryAPI from "services/entities/queries";
import { NotificationContext } from "context/notification";

import { getErrorReason, IApiError } from "interfaces/errors";
import {
  INVALID_PLATFORMS_FLASH_MESSAGE,
  INVALID_PLATFORMS_REASON,
} from "utilities/constants";
import { create } from "lodash";
import { ITeam, ITeamSummary } from "interfaces/team";

const baseClass = "save-as-new-query-modal";

interface ISaveAsNewQueryModal {
  router: InjectedRouter;
  initialQueryData: ICreateQueryRequestBody;
}

interface ISANQFormData {
  name: string;
  team: Partial<ITeamSummary>;
}

const SaveAsNewQueryModal = ({
  router,
  initialQueryData,
}: ISaveAsNewQueryModal) => {
  const { renderFlash } = useContext(NotificationContext);

  const [formData, setFormData] = useState<ISANQFormData>({
    name: `Copy of ${initialQueryData.name}`,
    team: {
      id: initialQueryData.team_id,
      name: undefined,
    },
  });

  const [isSaving, setIsSaving] = useState(false);
  // TODO - error state, 1 field
  // TODO - validation

  // take all existing data for query from parent, allow editing name and team
  const handleSave = () => async (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    setIsSaving(true);
    const {
      name,
      team: { id: teamId, name: teamName },
    } = formData;
    const createBody = {
      ...initialQueryData,
      name,
      team_id: teamId,
    };
    try {
      const response = await queryAPI.create(createBody);
      setIsSaving(false);
      renderFlash("success", `Successfully added query ${response.name}.`);
      router.push(
        getPathWithQueryParams(PATHS.QUERY_DETAILS(response.query.id), {
          team_id: response.query.team_id,
        })
      );
    } catch (createError: { data: IApiError }) {
      const reason = getErrorReason(createError);
      let errFlash = "Could not create query. Please try again.";
      if (reason.includes("already exists")) {
        let teamErrorText;
        if (createBody.team_id !== 0) {
          if (teamName) {
            teamErrorText = `the ${teamName} team`;
          } else {
            teamErrorText = "this team";
          }
        } else {
          teamErrorText = "all teams";
        }
        errFlash = `A query called "${createBody.name}" already exists for ${teamErrorText}.`;
      } else if (reason.includes(INVALID_PLATFORMS_REASON)) {
        errFlash = INVALID_PLATFORMS_FLASH_MESSAGE;
      }
      setIsSaving(false);
      renderFlash("error", errFlash);
    }
  };
  return <div className={`${baseClass}`}></div>;
};

export default SaveAsNewQueryModal;

// previous handler in EditQueryForm for reference

// const promptSaveAsNewQuery = () => (
//   evt: React.MouseEvent<HTMLButtonElement>
// ) => {
//   evt.preventDefault();

//   if (savedQueryMode && !lastEditedQueryName) {
//     return setErrors({
//       ...errors,
//       name: "Query name must be present",
//     });
//   }

//   let valid = true;
//   const { valid: isValidated } = validateQuerySQL(lastEditedQueryBody);

//   valid = isValidated;

//   if (valid) {
//     const newPlatformString = platformSelector
//       .getSelectedPlatforms()
//       .join(",") as CommaSeparatedPlatformString;

//     setIsSaveAsNewLoading(true);
//     const apiProps = {
//     };
//     queryAPI
//       .create({
//         name: lastEditedQueryName,
//         ...apiProps,
//       })
//       .then((response: { query: ISchedulableQuery }) => {
//         setIsSaveAsNewLoading(false);
//         router.push(
//           getPathWithQueryParams(PATHS.QUERY_DETAILS(response.query.id), {
//             team_id: response.query.team_id,
//           })
//         );
//         renderFlash("success", `Successfully added query.`);
//       })
//       .catch((createError: { data: IApiError }) => {
//         const createErrorReason = getErrorReason(createError);
//         if (createErrorReason.includes("already exists")) {
//           queryAPI
//             .create({
//               name: `Copy of ${lastEditedQueryName}`,
//               ...apiProps,
//             })
//             .then((response: { query: ISchedulableQuery }) => {
//               setIsSaveAsNewLoading(false);
//               router.push(
//                 getPathWithQueryParams(PATHS.EDIT_QUERY(response.query.id), {
//                   team_id: apiTeamIdForQuery,
//                 })
//               );
//               renderFlash(
//                 "success",
//                 `Successfully added query as "Copy of ${lastEditedQueryName}".`
//               );
//             })
//             .catch((createCopyError: { data: IApiError }) => {
//               if (
//                 getErrorReason(createCopyError).includes("already exists")
//               ) {
//                 let teamErrorText;
//                 if (apiTeamIdForQuery !== 0) {
//                   if (teamNameForQuery) {
//                     teamErrorText = `the ${teamNameForQuery} team`;
//                   } else {
//                     teamErrorText = "this team";
//                   }
//                 } else {
//                   teamErrorText = "all teams";
//                 }
//                 renderFlash(
//                   "error",
//                   `A query called "Copy of ${lastEditedQueryName}" already exists for ${teamErrorText}.`
//                 );
//               }
//               setIsSaveAsNewLoading(false);
//             });
//         } else if (createErrorReason.includes(INVALID_PLATFORMS_REASON)) {
//           setIsSaveAsNewLoading(false);
//           renderFlash("error", INVALID_PLATFORMS_FLASH_MESSAGE);
//         } else {
//           setIsSaveAsNewLoading(false);
//           renderFlash("error", "Could not create query. Please try again.");
//         }
//       });
//   }
// };
