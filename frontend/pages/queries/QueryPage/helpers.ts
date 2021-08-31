import permissionUtils from "utilities/permissions";
import { IUser } from "interfaces/user";

export const hasSavePermissions = (currentUser: IUser) => {
  return (
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser)
  );
};

export default { hasSavePermissions };
