import { useDispatch } from "react-redux";
// @ts-ignore
import { logoutUser } from "../../redux/nodes/auth/actions";

const LogoutPage = (): boolean => {
  const dispatch = useDispatch();

  dispatch(logoutUser());
  return false;
};

export default LogoutPage;
