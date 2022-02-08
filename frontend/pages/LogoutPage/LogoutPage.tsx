// @ts-ignore
import { clearToken } from "../../utilities/local"; // @ts-ignore

const LogoutPage = (): boolean => {
  clearToken();
  window.location.href = "/";
  return false;
};

export default LogoutPage;
