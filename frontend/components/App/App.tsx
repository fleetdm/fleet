import React, { useContext } from "react";
import { useDispatch, useSelector } from "react-redux";
import classnames from "classnames";

import { QueryClient, QueryClientProvider } from "react-query";

import { authToken } from "utilities/local"; // @ts-ignore
import { useDeepEffect } from "utilities/hooks"; // @ts-ignore
import { fetchCurrentUser } from "redux/nodes/auth/actions"; // @ts-ignore
import { getConfig, getEnrollSecret } from "redux/nodes/app/actions";
import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";
import TableProvider from "context/table";
import QueryProvider from "context/query";
import { AppContext } from "context/app1";

interface IAppProps {
  children: JSX.Element;
}

interface IRootState {
  auth: {
    user: IUser;
  };
}

const App = ({ children }: IAppProps) => {
  const dispatch = useDispatch();
  const { setCurrentUser, setConfig } = useContext(AppContext);
  const user = useSelector((state: IRootState) => state.auth.user);
  const queryClient = new QueryClient();

  useDeepEffect(() => {
    // on page refresh
    if (!user && authToken()) {
      dispatch(fetchCurrentUser()).catch(() => false);
    }

    if (user) {
      setCurrentUser(user);
      dispatch(getConfig())
        .then((config: IConfig) => setConfig(config))
        .catch(() => false);
      dispatch(getEnrollSecret()).catch(() => false);
    }
  }, [user]);

  const wrapperStyles = classnames("wrapper");
  return (
    <QueryClientProvider client={queryClient}>
      <TableProvider>
        <QueryProvider>
          <div className={wrapperStyles}>{children}</div>
        </QueryProvider>
      </TableProvider>
    </QueryClientProvider>
  );
};

export default App;
